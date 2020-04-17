package cmap

import (
	"encoding/json"
	"flag"
	Prop "github.com/magiconair/properties"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	cm "cmap"
	"strconv"
	"testing"
	"time"
)

type K8sConfMap struct {
	k8sNamespace   string
	k8sConfMap     string
	confMapKeyName string
	clientset      *kubernetes.Clientset
	confMap        *v1.ConfigMap
	confData       map[string]string
}

var ch chan *K8sConfMap

func getUnixTime(props *Prop.Properties) int64 {
	return getTime(props).Unix()
}

// Get localized time
func getTime(props *Prop.Properties) time.Time {
	location := props.GetString("splunkTimeZone", "")
	if location == "" {
		location = "UTC" //default to UTC
	}
	loc, _ := time.LoadLocation(location)
	now := time.Now().In(loc)
	return now
}

func cmapSetup(outClusterPtr *bool, props *Prop.Properties) *K8sConfMap {
	cmapName := props.GetString("cmapName", "")
	if cmapName == "" {
		log.Println("error: cmapName not defined")
		return nil
	}

	k8sConfMap := props.GetString("k8sConfMap", "")
	if k8sConfMap == "" {
		log.Println("error: k8sConfMap not defined")
		return nil
	}

	k8sNamespace := props.GetString("k8sNamespace", "")
	if k8sNamespace == "" {
		log.Println("error: k8sNamespace not defined")
		return nil
	}

	log.Println("timestamp:", getTime(props))
	utime := getUnixTime(props)
	confMapData := map[string]string{cmapName: strconv.FormatInt(utime, 10)}
	clientset := cm.SetupK8sClient(k8sNamespace, k8sConfMap, confMapData, outClusterPtr)

	if clientset != nil {
		//conf := cm.GetConfMap(k8sNamespace, k8sConfMap, clientset)

		return &K8sConfMap{k8sNamespace, k8sConfMap, cmapName, clientset, nil, confMapData}
	}
	return nil
}

func cmapGetConfMap(c *K8sConfMap) *K8sConfMap {
	conf := cm.GetConfMap(c.k8sNamespace, c.k8sConfMap, c.clientset)
	if conf != nil {
		c.confMap = conf
		return c
	}
	return nil
}

func TestSetupK8sClient(t *testing.T) {

	confFilePtr := flag.String("conf", "cmap_test.conf", "name of config file")
	flag.Parse()
	log.Println("using config file:", *confFilePtr)

	props := Prop.MustLoadFile(*confFilePtr, Prop.UTF8)

	outCluster := props.GetBool("outCluster", false)
	log.Println("out cluster:", outCluster)

	t.Run("T=1", func(t *testing.T) {
		log.Println("running T=1")
		conf := cmapSetup(&outCluster, props)
		ch = make(chan *K8sConfMap, 1)
		if conf == nil {
			t.Error("Expected ConfigMap, got", conf.confData)
		}
		ch <- conf
	})

    t.Run("T=2", func(t *testing.T) {
		log.Println("running T=2")
		conf := <-ch
		if conf != nil {
			confMap := cmapGetConfMap(conf)

			if confMap == nil {
				t.Error("Expected ConfigMap data got ", confMap.confData)
			}
			ch <- confMap
		}
	})

	t.Run("T=3", func(t *testing.T) {
		log.Println("running T=3")
		conf := <-ch
		if conf != nil {
			type ConfDataKv struct {
				Key string `json:"cmap-test"`
			}
			type ConfData struct {
				Key ConfDataKv `json:"data"`
			}
			utime := getUnixTime(props)
			data := []byte("{\"data\":{\"" + conf.confMapKeyName + "\":\"" + strconv.FormatInt(utime, 10) + "\"}}")
			cm.PutConfMap(data, conf.k8sNamespace, conf.k8sConfMap, conf.clientset)
			confMap := cmapGetConfMap(conf)
			var dat ConfData
			if err := json.Unmarshal(data, &dat); err != nil {
				panic(err)
			}
			for k := range confMap.confData {
				if confMap.confData[k] != dat.Key.Key {
					t.Error("Expected ", dat.Key.Key, " got ", confMap.confData[k])
				}
			}
		}
	})
}