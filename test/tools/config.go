package tools

import (
	"flag"
)

var Kubeconfig string
var Vmipaddr string

func init() {
	flag.StringVar(&Kubeconfig, "kubeconfig", "", "cluster kubeconfig")
	flag.StringVar(&Vmipaddr, "vmipaddr", "", "vm master ip address")
}

func FlagParse() {
	flag.Parse()

}
