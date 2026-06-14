package main

// knownExtensions 是已知官方扩展：子命令名 → module 路径。
//
// 未安装的已知扩展被调用时，goxctl 据此给出安装指引（仿 Azure CLI 的 dynamic install，
// 但用写死的注册表而非远程 index——团队扩展数量有限，简单可控）。新增官方扩展时在此登记。
var knownExtensions = map[string]string{
	"claude": "github.com/chinayin/goxctl-claude",
}

// installHint 返回已知扩展的 module 路径，用于未安装时给出安装指引。
func installHint(name string) (string, bool) {
	mod, ok := knownExtensions[name]
	return mod, ok
}
