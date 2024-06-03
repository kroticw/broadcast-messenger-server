package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var cfgFile string

type configuration struct {
	Listen       string `mapstructure:"listen"`
	LogLevel     uint   `mapstructure:"logLevel"`
	BaseURL      string `mapstructure:"baseUrl"`
	PlatformName string `mapstructure:"platformName"`

	OpenTelemetry *struct {
		Enabled     bool    `mapstructure:"enabled"`
		ServiceName string  `mapstructure:"serviceName"`
		Endpoint    string  `mapstructure:"endpoint"`
		TraceRate   float64 `mapstructure:"traceRate"`
	} `mapstructure:"openTelemetry"`
}

var cfg configuration

var logger *logrus.Logger

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "shelezyaka",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initLogger, initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.shelezyaka.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigType("yaml")
		viper.SetConfigName("config.yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		logger.WithField("filename", viper.ConfigFileUsed()).Infoln("Using config file")
	}

	err := viper.Unmarshal(&cfg)
	if err != nil {
		logger.WithError(err).Fatalln("Ошибка парсинга конфига")
	}

	if cfg.LogLevel > 0 {
		logger.SetLevel(logrus.Level(cfg.LogLevel))
	}
}

func initLogger() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyMsg: "log",
		},
	})
}
