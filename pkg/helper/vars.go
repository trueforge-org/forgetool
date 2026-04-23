package helper

import (
	"os"
	"path/filepath"
)

var (
	UserCacheDir, _ = os.UserCacheDir()
	CacheDir        = filepath.Join(UserCacheDir, "forgetool")
	HelmCache       = filepath.Join(CacheDir, "tgz_cache")
	KubeCache       = filepath.Join(CacheDir, "kubernetes")
	BaseCache       = filepath.Join(CacheDir, "base")
	RootCache       = filepath.Join(CacheDir, "root")
	PatchCache      = filepath.Join(CacheDir, "patches")
	DocsCache       = filepath.Join(CacheDir, "docs")

	IndexCache = "./index_cache"
	GpgDir     = ".cr-gpg"
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
