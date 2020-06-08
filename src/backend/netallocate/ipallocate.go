package netallocate

import (
	"fmt"
	"github.com/containernetworking/cni/pkg/types/current"
	"net"
	"strconv"
	"strings"
	"util/log"
	"util/etcdclient"
)

func IpAllocate(ipGroup string, ipRange []string) (string, string, error) {
	/* 从ip范围内选择可以使用的IP地址
	   逻辑待补充，ip格式为1.1.1.1/23 */
	key := "/registry/" + ipGroup + "/iprange"
	podCfgIpRange, err := etcdclient.Etcdclient.Get(key)
	if err != nil {
		log.Errorf("在etcd中无IP段信息，无法分配IP地址")
		return "", "", err
	}
	log.Debugln(podCfgIpRange)

	podCfgIpList := strings.Split(podCfgIpRange, ",")
        if len(podCfgIpList) == 0 {
		return "", "", fmt.Errorf("没有可用的地址段")
	}
        configIp := podCfgIpList[0]
        
	log.Infof("分配到IP地址: %s",configIp)


	/* 根据IP拉取网关信息，这边基于/23位子网掩码进行计算*/
	netAB := strings.Join(strings.Split(strings.Split(configIp, "/")[0], ".")[0:3], ".")
	netC := strings.Split(strings.Split(configIp, "/")[0], ".")[3]
	netCInt, err := strconv.Atoi(netC)
	if err != nil {
		log.Errorf("IP: %s, 网关地址数据类型转换失败",configIp )
		return "", "", fmt.Errorf("网关地址数据类型转换失败")
	}
	if netCInt%2 != 0 {
		configGw := strings.Join(strings.Split(strings.Split(configIp, "/")[0], ",")[0:2], ".") + "2/24"
		log.Infof("IP: %s, 分配的网关地址为: %s",configIp, configGw)
		return configIp, configGw, nil
	}
	netCStr := strconv.Itoa(netCInt - 1)
	configGw := netAB + "." + netCStr + ".2/24"
	log.Infof("IP: %s, 分配的网关地址为: %s",configIp, configGw)

	var ret []string
	for _, val := range podCfgIpList {
		if val != configIp {
			ret = append(ret, val)
		}
	}
	
        podCfgIpStr := strings.Join(ret, ",")
        log.Infof(podCfgIpStr)
	err = etcdclient.Etcdclient.Put(key,podCfgIpStr)
	if err != nil {
		return "", "", fmt.Errorf("更新IpRange失败")
	}

	return configIp, configGw, nil
}

func VlanAllocate(ip string) int {
	/* 根据IP地址获取VLAN
	   这里写死，实际需要调用内部平台做hash匹配 */
	vlanMap := make(map[string]int)
	vlanMap["192.168.1.1/23"] = 2135
	vlanId := vlanMap[ip]
	return vlanId
}

func IpCfgConv(configIp, configGw string) (*current.IPConfig, error) {
	_, configIpNet, err := net.ParseCIDR(configIp)
	if err != nil {
		return nil, fmt.Errorf("解析IP地址到ipnet失败")
	}
	configGwIp, _, err := net.ParseCIDR(configGw)
	if err != nil {
		return nil, fmt.Errorf("解析GW到netip失败")
	}
	return &current.IPConfig{
		Version: "4",
		Address: *configIpNet,
		Gateway: configGwIp,
	}, nil
}
