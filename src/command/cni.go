package main

import (
	"backend/netallocate"
	"backend/portmanagement"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ns"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"strconv"
	"strings"
	"util/config"
	"util/log"
	"util/etcdclient"
)

type NetConf struct {
	types.NetConf
	Master string
	Mode   string
	MTU    int
}

func loadConf(bytes []byte) (*NetConf, string, error) {
	n := &NetConf{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, "", fmt.Errorf("failed to load netconf: %v", err)
	}
	n.MTU = 1500
	return n, n.CNIVersion, nil
}

func getIpRange(podNameSpace, podName string) ([]string, string, error) {
	var (
		k8sconfig = flag.String("kubeconfig", "/root/.kube/config", "admin kubeconfig")
		config    *rest.Config
		err       error
	)
	flag.Parse()
	config, err = clientcmd.BuildConfigFromFlags("", *k8sconfig)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to Get Kubeconfig, %v", err)
	}
	log.Debugln("获取KubeConfig 配置文件成功")
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to Reslov Kubeconfig, %v", err)
	}
	log.Debugln("解析KubeConfig成功, 生产client对象完成")

	pods, err := clientSet.CoreV1().Pods(podNameSpace).List(metav1.ListOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("Failed to List Pods, %s", err.Error())
	}
	//log.Debugf("NameSpace: %s 下存在以下Pods: %s", podNameSpace, pods)

	for _, pod := range pods.Items {
		if podName != pod.ObjectMeta.Name {
			continue
		}
		ipAnnotation := pod.Annotations["ipv4list"]
		ipGroup := pod.Annotations["ipgroupname"]
		ipAnnotationList := strings.Split(ipAnnotation, ",")
		log.Debugf("Podname: %s, ipv4亲和性列表: %s",pod.ObjectMeta.Name, ipAnnotationList)
		return ipAnnotationList, ipGroup, nil
	}
	log.Errorf("YAML不存在ipv4的Annotations, Podname: %s",podName )
	return nil, "", fmt.Errorf("Can not Find IPRange of Pod %s", podName)
}

func loadArgMap(envArgs string) (map[string]string, error) {
	log.Infof("envArgs string is :%s",envArgs)
	argsMap := make(map[string]string)
	pairs := strings.Split(envArgs, ";")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("Reslov IPAM Args Failed")
		}
		keyString := kv[0]
		valueString := kv[1]
		argsMap[keyString] = valueString
	}
	return argsMap, nil
}

func main() {
	var confPath = flag.String("confPath", "/etc/cni/conf/default.ini", "load conf file")
	flag.Parse()

	// 配置文件初始化
	config.GlobalConf.CfgInit(*confPath)

	// 日志初始化
	log.InitLog()

	// etcd初始化
	var etcdCluster = []string{"192.168.159.145:2379"}
	etcdclient.ClientInitWitchCA("/opt/k8s/work/etcd.pem", "/opt/k8s/work/etcd-key.pem", "/opt/k8s/work/ca.pem", 4, 4, 60, etcdCluster)
	// 加载插件本体
	skel.PluginMain(cmdAdd, cmdGet, cmdDel, version.All, "todo")
}

func cmdAdd(args *skel.CmdArgs) error {
	// 获取namespache
	netNS, err := ns.GetNS(args.Netns)
	if err != nil {
		log.Errorln("获取namespache对象失败")
		return fmt.Errorf("Get NameSpace Object Failed")
	}
	log.Infof("获取NameSpace对象成功, 分配到pause containerid: %s, 对应namespace 路径为: %s", args.ContainerID, netNS.Path())
	// 解析环境参数
	argsMap, err := loadArgMap(args.Args)
	if err != nil {
		log.Errorln("解析EnvArgs参数失败")
		return err
	}
	podName := argsMap["K8S_POD_NAME"]
	podNameSpace := argsMap["K8S_POD_NAMESPACE"]
	log.Infof("待创建的Pod: %s, 所在的K8S NameSpace: %s", podName, podNameSpace)

	// 分配IP，逻辑根据业务场景制定
	ipRange, ipGroup, err := getIpRange(podNameSpace, podName)
	if err != nil {
		log.Errorf("根绝Podname获取IP范围失败")
		return err
	}

	log.Infof("PodName: %s, 将从列表: %s 中获取IP地址", podName, ipRange)
	// 获取IP和网关信息,逻辑根据业务场景制定
	configIp, configGw, err := netallocate.IpAllocate(ipGroup,ipRange)
	if err != nil {
		log.Errorln("获取PodIP 以及网关IP失败")
		return err
	}
	log.Infof("Pod: %s, 分配IP: %s, 网关: %s", podName, configIp, configGw)

	// 根据IP获得vlanid
	vlanId := netallocate.VlanAllocate(configIp)

	// 获取归属bond子接口以及网桥,产线默认bond1
	vlanIdStr := strconv.Itoa(vlanId)
	businessInt := config.GlobalConf.GetStr("server", "businessint")
	subBondName := businessInt + "." + vlanIdStr
	bridgeName := "br" + vlanIdStr
	log.Infof("Pod: %s, 所属VLAN: %s, 子接口: %s, 网桥: %s", podName, vlanIdStr, subBondName, bridgeName)

	// 创建网桥
	bridgeObject := portmanagement.NewBridgeObject(bridgeName)
	br, err := bridgeObject.Create()
	if err != nil {
		log.Errorf("创建网桥失败, 错误信息: %s", err.Error())
		return err
	}
	log.Infof("创建网桥完成, 创建接口: %s", bridgeObject.Name)

	// 创建子接口
	vlanObject := portmanagement.NewVlanObject(businessInt, subBondName, br, vlanId)
	_, err = vlanObject.Create()
	if err != nil {
		log.Errorf("创建vlan port 失败，错误信息: %s", err.Error())
		return err
	}
	log.Infof("创建子接口完成, 创建接口: %s", subBondName)

	// 创建veth
	vethObject := portmanagement.NewVethObject("eth0", netNS.Path(), configIp, configGw)
	localIfname, err := vethObject.Create()
	if err != nil {
		log.Errorf("创建veth失败, 错误信息: %s", err.Error())
		return err
	}
	log.Infof("创建veth完成, 创建接口: %s", localIfname)

	// veth 挂载到网桥
	err = vethObject.Attach(bridgeObject.Name, configIp)
	if err != nil {
		log.Errorf("Veth: %s 挂载到网桥: %s 失败", localIfname, bridgeObject.Name)
		return err
	}
	log.Infof("Veth: %s 挂载到网桥: %s 成功", localIfname, bridgeObject.Name)

	// 定义返回
	result := &current.Result{}
	ipc, err := netallocate.IpCfgConv(configIp, configGw)
	if err != nil {
		log.Errorf("解析result ipc 失败")
	}
	result.IPs = append(result.IPs, ipc)
	_, cniVersion, err := loadConf(args.StdinData)
	if err != nil {
		log.Errorf("获取CNI版本失败")
		return err
	}
	return types.PrintResult(result, cniVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	log.Infoln("开始调用cmd delete...")
	return nil
}

func cmdGet(args *skel.CmdArgs) error {
	log.Infoln("开始调用cmd get...")
	return nil
}
