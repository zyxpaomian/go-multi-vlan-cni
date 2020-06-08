package portmanagement

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"net"
)

type Bridge struct {
	Name string
}

func NewBridgeObject(bridgeName string) *Bridge {
	return &Bridge{
		Name: bridgeName,
	}
}

func JudgeExist(intName string) bool {
	_, err := net.InterfaceByName(intName)
	// 判断接口是否存在，存在则返回true
	if err == nil {
		return true
	} else {
		return false
	}
}

func (b *Bridge) Create() (*netlink.Bridge, error) {
	if JudgeExist(b.Name) == false {
		bridge := &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name:   b.Name,
				MTU:    1500,
				TxQLen: -1,
			},
		}
		//创建bridge接口
		if err := netlink.LinkAdd(bridge); err != nil {
			return nil, fmt.Errorf("Create Bridge Interface: %s Failed, ErrorInfo: %s", b.Name, err.Error())
		}

		//打开bridge接口
		if err := netlink.LinkSetUp(bridge); err != nil {
			return nil, fmt.Errorf("SetUp Bridge Interface: %s Failed, ErrorInfo: %s", b.Name, err.Error())
		}
		return bridge, nil
	} else {
		bridge := &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name:   b.Name,
				MTU:    1500,
				TxQLen: -1,
			},
		}
		return bridge, nil
	}

}
