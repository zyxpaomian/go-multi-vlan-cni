package portmanagement

import (
	"fmt"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/j-keck/arping"
	"github.com/vishvananda/netlink"
	"net"
)

type Veth struct {
	HostIfName      string
	ContainerIfName string
	NetNs           string
	ContainerIp     string
	ContainerGw     string
}

func NewVethObject(containerIfName, nsPath, containerIp, containerGw string) *Veth {
	return &Veth{
		ContainerIfName: containerIfName,
		NetNs:           nsPath,
		ContainerIp:     containerIp,
		ContainerGw:     containerGw,
	}
}

func (e *Veth) Create() (string, error) {
	netns, err := ns.GetNS(e.NetNs)
	if err != nil {
		return "", fmt.Errorf("Get NetNs Falied, Failed NetNs Path: %s", e.NetNs)
	}

	// 创建veth并获取一堆接口对象的方法
	var handler = func(hostNS ns.NetNS) error {
		hostVeth, containerVeth, err := ip.SetupVeth(e.ContainerIfName, 1500, hostNS)
		if err != nil {
			return fmt.Errorf("Create Veth: %s Failed On NetNs: %s", e.ContainerIfName, e.NetNs)
		}
		e.HostIfName = hostVeth.Name
		// 解析container的接口IP
		containerIp, containerNet, err := net.ParseCIDR(e.ContainerIp)
		if err != nil {
			return fmt.Errorf("Reslov Container IP: %s  Failed", e.ContainerIp)
		}

		containerNet.IP = containerIp
		// 获取container int对象
		containerLink, err := netlink.LinkByName(containerVeth.Name)
		if err != nil {
			return fmt.Errorf("Get Container IP Failed, Container IP: %s", containerVeth.Name)
		}

		// 配置container IP地址
		containerIpaddr := &netlink.Addr{IPNet: containerNet, Label: ""}
		if err = netlink.AddrAdd(containerLink, containerIpaddr); err != nil {
			return fmt.Errorf("Set Container Interface IP Failed")
		}

		// 配置container 网关&&路由
		containerGwIp, containerGwNet, err := net.ParseCIDR(e.ContainerGw)
		containerGwNet.IP = containerGwIp
		_, defaultDst, _ := net.ParseCIDR("0.0.0.0/0")
		defaultRoute := &netlink.Route{
			LinkIndex: containerLink.Attrs().Index,
			Gw:        containerGwNet.IP,
			Dst:       defaultDst,
		}
		err = netlink.RouteAdd(defaultRoute)
		if err != nil {
			return fmt.Errorf("Add Container Default Route Failed")
		}
		return nil
	}

	err = netns.Do(handler)
	if err != nil {
		return "", fmt.Errorf("Config Veth Pair Interface Failed")
	}
	defer netns.Close()
	return e.HostIfName, nil
}

func (e *Veth) Attach(brName, configIp string) error {
	hostLink, _ := netlink.LinkByName(e.HostIfName)
	hostBridgeLink, _ := netlink.LinkByName(brName)
	hostBridgeLinkIndex := hostBridgeLink.Attrs().Index
	if err := netlink.LinkSetMasterByIndex(hostLink, hostBridgeLinkIndex); err != nil {
		return fmt.Errorf("Attach HostLink To Bridge Failed")
	}

	netNS, err := ns.GetNS(e.NetNs)
	if err != nil {
		return fmt.Errorf("Get NetNs Falied, Failed NetNs Path: %s", e.NetNs)
	}

	var handler = func(hostNS ns.NetNS) error {
		hostInterface, err := net.InterfaceByName("eth0")
		if err != nil {
			return fmt.Errorf("Get NS Veth Interface Object Failed, Failed NS Path: %s", e.NetNs)
		}
		srcIp, _, err := net.ParseCIDR(configIp)
		if err != nil {
			return fmt.Errorf("Get Local Veth IP Object Failed")
		}
		if err = arping.GratuitousArpOverIface(srcIp, *hostInterface); err != nil {
			return fmt.Errorf("Send Arp BoardCase Failed On NS Path : %s", e.NetNs)
		}
		return nil
	}
	if err := netNS.Do(handler); err != nil {
		return err
	}
	defer netNS.Close()
	return nil

}
