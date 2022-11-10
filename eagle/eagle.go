package eagle

// const (
// 	AssetsDirectory  string = "assets"
// 	ContentDirectory string = "content"
// )

// type Eagle struct {
// 	notifier.Notifier
// 	*fs.FS

// 	Cache *cache.Cache

// 	log    *zap.SugaredLogger
// 	DB     database.Database
// 	Config *config.Config
// }

// func NewEagle(conf *config.Config) (*Eagle, error) {
// 	var srcSync fs.FSSync
// 	if conf.Development {
// 		srcSync = fs.NewPlaceboSync()
// 	} else {
// 		srcSync = fs.NewGitSync(conf.Source.Directory)
// 	}
// 	srcFs := fs.NewFS(conf.Source.Directory, srcSync)

// 	e := &Eagle{
// 		log:    log.S().Named("eagle"),
// 		FS:     srcFs,
// 		Config: conf,
// 	}

// 	if conf.Notifications.Telegram != nil {
// 		notifications, err := notifier.NewTelegramNotifier(conf.Notifications.Telegram)
// 		if err != nil {
// 			return nil, err
// 		}
// 		e.Notifier = notifications
// 	} else {
// 		e.Notifier = notifier.NewLogNotifier()
// 	}

// 	var err error

// 	e.DB, err = database.NewDatabase(&conf.PostgreSQL)
// 	if err != nil {
// 		return nil, err
// 	}

// 	go e.indexAll()
// 	return e, nil
// }

// func (e *Eagle) Close() {
// 	if e.DB != nil {
// 		e.DB.Close()
// 	}
// }
