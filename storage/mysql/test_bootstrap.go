package mysql

import (
	"code.uber.internal/go-common.git/x/log"
	"fmt"
	"github.com/mattes/migrate/migrate"
	"os"
	"path"
)

func downSync(cfg *Config) []error {
	connString := cfg.MigrateString()
	errors, ok := migrate.DownSync(connString, cfg.Migrations)
	if !ok {
		return errors
	}
	return nil
}

// LoadConfigWithDB instantiates a config with a DB connection
func LoadConfigWithDB() *Config {
	conf := &Config{
		User:       "peloton",
		Password:   "peloton",
		Host:       "127.0.0.1",
		Port:       8193,
		Database:   "peloton",
		Migrations: "migrations",
	}
	dir := os.Getenv("UBER_CONFIG_DIR")
	if dir != "" {
		conf.Migrations = path.Join(dir, "..", "storage", "mysql", conf.Migrations)
	}
	fmt.Printf("dir=%s %s", dir, conf.Migrations)
	err := conf.Connect()
	if err != nil {
		panic(err)
	}
	if errs := downSync(conf); errs != nil {
		log.Warnf(fmt.Sprintf("downSync is having the following error: %+v", errs))
	}

	// bring the schema up to date
	if errs := conf.AutoMigrate(); errs != nil {
		panic(fmt.Sprintf("%+v", errs))
	}
	fmt.Println("setting up again")
	return conf
}