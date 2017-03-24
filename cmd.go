package cmd

import (
	"os/exec"
	"time"
	"bytes"
	"errors"
	"syscall"
	"strconv"
)

var (
	ErrTimeout = errors.New("exec timeout")
)

type FixedBuffer struct {
	Buf     bytes.Buffer
	Size    int
	CurSize int
}

func NewFixedBuffer (size int) *FixedBuffer {
	return &FixedBuffer{Buf:bytes.Buffer{}, Size:size}	
}

func (this *FixedBuffer) Write(p []byte) (n int, err error) {
	if this.Size < 0 {
		n, err := this.Buf.Write(p)
		this.CurSize += n
		return n, err
	} else {
		if this.CurSize >= this.Size {
			return len(p), nil
		}
		var toWrite []byte
		if this.CurSize + len(p) >= this.Size {
			diff := this.Size - this.CurSize
			toWrite = p[0:diff]
		} else {
			toWrite = p
		}
		n, err = this.Buf.Write(toWrite)
		this.CurSize += n
		return n, err
	}
}

//ExecWithTimeout           执行shell命令
//param cmdstr:             命令
//param user:               以指定用户执行命令
//param timeout:            执行命令的超时时间
//param limit:              标准输出、标准出错返回内容的大小限制，超过被截断
//param isCombineStderr:    是否包括标准出错内容
//param isPrefixTimeoutCmd: 若timeout>0,且isAppendTimeoutCmd为true,则自动添加timeout到命令前,
//                          否则可自定义timeout的位置
func ExecWithTimeout(cmdstr string, user string, timeout int, limit int, isCombineStderr bool, isPrefixTimeoutCmd bool) (string, error) {
	if timeout == 0 {
		return "", ErrTimeout
	} 
	if timeout > 0 {
		if isPrefixTimeoutCmd {
			cmdstr = "timeout " + strconv.Itoa(timeout) + " " + cmdstr
		}
	}
	var err error
	done := make(chan error)
	fixedbuf := NewFixedBuffer(limit)
	command := exec.Command("su", "-c", cmdstr, user)
	command.Stdout = fixedbuf
	if isCombineStderr {
		command.Stderr = fixedbuf
	}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid:true}
	command.Start()

	go func() {
		done <- command.Wait()
	}()
	
	if timeout > 0 {
		select {
        	case <-time.After(time.Duration(timeout) * time.Second):
				go func() {
					<-done
				}()
				err = ErrTimeout
			case <-done:
				err = nil
		}
	} else {
		select {
			case <-done:
				err = nil
		}
	}

	output := fixedbuf.Buf.String()
	return output, err
}
