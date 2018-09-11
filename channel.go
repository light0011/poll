package light_poll

import (
	"time"
	"sync"
	"errors"
	"fmt"
)

type IdleConn struct {
	conn interface{}
	t time.Time
}


type PoolConfig struct {
	//连接池最小连接数
	MinCap int
	//连接池最大连接数
	MaxCap int
	//生成链接的方法
	Open func()(interface{}, error)
	//关闭连接的方法
	Close func(interface{}) error
	//空闲时间
	IdleTimeOut time.Duration
}

type channelPoll struct {
	sync.Mutex
	conns chan *IdleConn
	open func()(interface{},error)
	close func(interface{}) error
	idleTimeOut time.Duration
}

func NewChannelPool(poolConfig *PoolConfig)(Pool,error)  {
	if poolConfig.MinCap <0 || poolConfig.MaxCap <=0 || poolConfig.MinCap > poolConfig.MaxCap{
		return nil,errors.New("invalid setting")

	}

	c := &channelPoll{
		conns:	make(chan *IdleConn,poolConfig.MaxCap),
		open:	poolConfig.Open,
		close:	poolConfig.Close,
		idleTimeOut:	poolConfig.IdleTimeOut,
	}

	for i:=0;i< poolConfig.MinCap ;i++  {
		conn,err := c.open()
		if err!= nil {
			c.Release()
			return  nil,fmt.Errorf("factory is not able to fill the pool: %s", err)
		}
		c.conns <- &IdleConn{conn: conn, t: time.Now()}
	}

	return c,nil

}

func (c * channelPoll)getConns() chan *IdleConn  {
	c.Lock()
	conns := c.conns
	c.Unlock()
	return conns
}
func (c *channelPoll) Get()(interface{},error)  {
	conns := c.getConns()
	if conns == nil {
		return nil,ErrClosed
	}
	for  {
		select {
		case selectConn := <-conns:
			if selectConn == nil{
				if timeout := c.idleTimeOut;timeout >0 {
					if selectConn.t.Add(timeout).Before(time.Now()) {
						c.close(selectConn.conn)
						continue
					}
				}
				return selectConn.conn,nil
			}
		default:
			conn,err := c.open()
			if err != nil {
				return  nil,err
			}
			return conn,nil

		}
	}

}

func (c * channelPoll)Put(conn interface{}) error {
	if conn == nil{
		return errors.New("conn is nul")
	}
	c.Lock()
	if c.conns == nil {
		return c.close(conn)
	}
	select {
	case c.conns <- &IdleConn{conn:conn,t:time.Now()}:
		return nil
	default:
		return c.close(conn)
	}


}

func (c * channelPoll)Close(conn interface{}) error  {
	if conn == nil {
		return errors.New("conn is nil")
	}
	return c.close(conn)
}




func (c * channelPoll) Release()  {
	c.Lock()
	conns := c.conns
	c.conns=nil
	c.open=nil
	closeFun := c.close
	c.close=nil
	c.Unlock()

	if conns == nil{
		return
	}

	close(conns)
	for everyConn := range conns  {
		closeFun(everyConn.conn)
	}
}

func (c * channelPoll)Len()  int{
	return len(c.getConns())
}

