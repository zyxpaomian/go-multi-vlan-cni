package portmanagement

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"util/log"
)

type Vlan struct {
	ParentName string
	Name       string
	MasterBr   *netlink.Bridge
	VlanId     int
}

func NewVlanObject(parentInterfaceName, interfaceName string, br *netlink.Bridge, vlanid int) *Vlan {
	return &Vlan{
		ParentName: parentInterfaceName,
		Name:       interfaceName,
		MasterBr:   br,
		VlanId:     vlanid,
	}
}

func (v *Vlan) Create() (*netlink.Vlan, error) {
	if JudgeExist(v.ParentName) == false {
		return nil, fmt.Errorf("Parent Interface: %s Not Existed", v.ParentName)
	}
	parentLink, _ := netlink.LinkByName(v.ParentName)
	parentLinkIndex := parentLink.Attrs().ParentIndex

	if JudgeExist(v.Name) == false {
		vlan := &netlink.Vlan{
			LinkAttrs: netlink.LinkAttrs{
				Name:        v.Name,
				MTU:         1500,
				TxQLen:      -1,
				ParentIndex: parentLinkIndex,
				MasterIndex: v.MasterBr.Attrs().Index,
			},
			VlanId: v.VlanId,
		}
		//创建vlan接口
		if err := netlink.LinkAdd(vlan); err != nil {
			log.Errorf("Create Vlan Interface: %s Failed, ErrorInfo: %s", v.Name, err.Error())
			return nil, fmt.Errorf("Create Vlan Interface: %s Failed, ErrorInfo: %s", v.Name, err.Error())
		}

		//打开vlan接口
		if err := netlink.LinkSetUp(vlan); err != nil {
			log.Errorf("SetUp Vlan Interface: %s Failed, ErrorInfo: %s", v.Name, err.Error())
			return nil, fmt.Errorf("SetUp Vlan Interface: %s Failed, ErrorInfo: %s", v.Name, err.Error())
		}
		return vlan, nil
	} else {
		vlan := &netlink.Vlan{
			LinkAttrs: netlink.LinkAttrs{
				Name:        v.Name,
				MTU:         1500,
				TxQLen:      -1,
				ParentIndex: parentLinkIndex,
				MasterIndex: v.MasterBr.Attrs().Index,
			},
			VlanId: v.VlanId,
		}
		return vlan, nil
	}
	return nil, fmt.Errorf("Create Vlan Interface: %s Failed, ErrorInfo: Unknown Error", v.Name)
}
