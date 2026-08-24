package timing

import "os"

func osStatImpl(p string) (any, error) { return os.Stat(p) }
