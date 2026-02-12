package helper

import (
	"os"
	"path/filepath"
)

var (
	HelmCache       = filepath.Join(CacheDir, "tgz_cache")
	UserCacheDir, _ = os.UserCacheDir()
	TalEnv          = make(map[string]string)
	ClusterName     = "main"
	KubeCache       = filepath.Join(CacheDir, "kubernetes")
	BaseCache       = filepath.Join(CacheDir, "base")
	RootCache       = filepath.Join(CacheDir, "root")
	PatchCache      = filepath.Join(CacheDir, "patches")
	DocsCache       = filepath.Join(CacheDir, "docs")
	CacheDir        = filepath.Join(UserCacheDir, "forgetool")
	ClusterPath     = filepath.Join("./clusters", ClusterName)
	ClusterEnvFile  = filepath.Join(ClusterPath, "/clusterenv.yaml")
	TalConfigFile   = filepath.Join(ClusterPath, "/talos", "talconfig.yaml")
	TalosPath       = filepath.Join(ClusterPath, "/talos")
	KubernetesPath  = filepath.Join(ClusterPath, "/kubernetes")
	TalosGenerated  = filepath.Join(TalosPath, "/generated")
	TalosConfigFile = filepath.Join(TalosGenerated, "talosconfig")
	TalSecretFile   = filepath.Join(TalosGenerated, "talsecret.yaml")
	AllIPs          = []string{}
	ControlPlaneIPs = []string{}
	WorkerIPs       = []string{}

	IndexCache = "./index_cache"
	GpgDir     = ".cr-gpg" // Adjust the path based on your project structure
	Logo       = `

  _______              ______                   
 |__   __|            |  ____|                  
    | |_ __ _   _  ___| |__ ___  _ __ __ _  ___ 
    | | '__| | | |/ _ \  __/ _ \| '__/ _` + "`" + ` |/ _ \
    | | |  | |_| |  __/ | | (_) | | | (_| |  __/
    |_|_|   \__,_|\___|_|  \___/|_|  \__, |\___|
                                      __/ |     
        ____                ______   |___/  __      
       / __/__  _______ ___/_  __/__  ___  / /
      / _// _ \/ __/ _ ` + "`" + `/ -_) / / _ \/ _ \/ / 
     /_/  \___/_/  \_, /\__/_/  \___/\___/_/  
                  /___/                       
                                     
`
)
