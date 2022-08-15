package tools

import (
	"flag"
)

var Kubeconfig string
var Vmipaddr string
var Workeripaddr string

func init() {
	flag.StringVar(&Kubeconfig, "kubeconfig", "", "cluster kubeconfig")
	flag.StringVar(&Vmipaddr, "vmipaddr", "", "vm master ip address")
	flag.StringVar(&Workeripaddr, "vmipaddr2", "", "vm worker ip address")
}

func FlagParse() {
	flag.Parse()

}
