package cfg

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Bytes []byte

type DatabaseConfig struct {
	DBName          string        `yaml:"db_name"`
	Host            string        `env:"MYSQL_HOST, required"`
	Username        string        `env:"MYSQL_USER, required"`
	Password        string        `env:"MYSQL_PASSWORD, required"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime"`
}

type MongoConfig struct {
	URI         string        `env:"MONGO_URI, required"`
	MaxPoolSize uint64        `yaml:"max_pool_size"`
	MinPoolSize uint64        `yaml:"min_pool_size"`
	MaxIdletime time.Duration `yaml:"max_idle_time"`
	Database    string        `yaml:"database"`
	Collection  string        `yaml:"collection"`
}

type ServerConfig struct {
	ReadWriteTimeout time.Duration `yaml:"read_write_timeout"`
	IdleTimeout      time.Duration `yaml:"idle_timeout"`
	Address          string        `yaml:"address"`
	TrustedProxies   string        `env:"TRUSTED_PROXIES, required"`
}

type MetaConfig struct {
	Name        string `yaml:"api_name" json:"api"`
	Description string `yaml:"api_description" json:"description"`
	Version     string `yaml:"api_version" json:"version"`
	VersionTag  string `json:"version_tag"`
	URL         string `yaml:"api_url" json:"url"`
	Group       string `yaml:"group" json:"-"`
}

type LogConfig struct {
	Level              string        `yaml:"level"`
	JSONFormat         bool          `yaml:"json_format"`
	SlowQueryThreshold time.Duration `yaml:"slow_query_theshold"`
	DBLevel            string        `yaml:"db_level"`
	LogCaller          bool          `yaml:"log_caller"`
}

type Models struct {
	Path     string `yaml:"path"`
	MaxDoses int    `yaml:"max_doses"`
}

type RConfig struct {
	RScriptPathWin   string `yaml:"rscript_path_win"`
	RScriptPathUnix  string `yaml:"rscript_path_unix"`
	DoseAdjustScript string `yaml:"dose_adjust_script"`
	RWorker          int    `yaml:"r_worker"`
}

type AuthTokenConfig struct {
	Secret                Bytes         `env:"JWT_SECRET, required"`
	AccessExpirationTime  time.Duration `yaml:"access_expiration_time"`
	RefreshExpirationTime time.Duration `yaml:"refresh_expiration_time"`
	Issuer                string        `yaml:"issuer"`
}

type ResetTokenConfig struct {
	ExpirationTime time.Duration `yaml:"expiration_time"`
	RetryInterval  time.Duration `yaml:"retry_interval"`
}

type MedInfoConfig struct {
	URL             string        `yaml:"url"`
	ExpiryThreshold time.Duration `yaml:"expiry_threshold"`
	Login           string        `env:"ABDATA_LOGIN, required"`
	Password        string        `env:"ABDATA_PASSWORD, required"`
}

type SchemaConfig struct {
	PreCheck string `yaml:"precheck"`
}

type MailerConfig struct {
	SendEmail string `env:"SEND_EMAIL, required"`
	APIKey    string `env:"SEND_EMAIL_API_KEY, required"`
}

type MMCConfig struct {
	ResultEndpoint  string        `yaml:"result_endpoint"`
	AuthEndpoint    string        `yaml:"auth_endpoint"`
	Interval        time.Duration `yaml:"fetch_interval"`
	BatchSize       int           `yaml:"batch_size"`
	ExpiryThreshold time.Duration `yaml:"expiry_threshold"`
	PDFPrefix       string        `yaml:"pdf_prefix"`
	Login           string        `env:"MMC_LOGIN, required"`
	Password        string        `env:"MMC_PASSWORD, required"`
}

type JobRunnerConfig struct {
	Interval time.Duration `yaml:"fetch_interval"`
	Timeout  time.Duration `yaml:"timeout"`
	MaxJobs  int           `yaml:"max_concurrent_jobs"`
}

func (b *Bytes) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var secret string
	if err := unmarshal(&secret); err != nil {
		return err
	}
	*b = []byte(secret)
	return nil
}

type APIConfig struct {
	Meta       MetaConfig       `yaml:"meta"`
	Server     ServerConfig     `yaml:"server"`
	RLang      RConfig          `yaml:"rlang"`
	JobRunner  JobRunnerConfig  `yaml:"job_runner"`
	Database   DatabaseConfig   `yaml:"database"`
	Mongo      MongoConfig      `yaml:"mongo"`
	Log        LogConfig        `yaml:"log"`
	AuthToken  AuthTokenConfig  `yaml:"auth_token"`
	ResetToken ResetTokenConfig `yaml:"reset_token"`
	MedInfoAPI MedInfoConfig    `yaml:"medinfo"`
	Schema     SchemaConfig     `yaml:"schema"`
	Models     Models           `yaml:"models"`
	MMCAPI     MMCConfig        `yaml:"mmc"`
	Mailer     MailerConfig     // TODO: kill this and the endpoints that are not used
}

// Read reads the configuration file and environment variables
// and returns the APIConfig struct or an error.
// ConfigFile cannot have any unknown fields.
// Environment variables cannot be missing.
func MustParseYAML(configFile string) *APIConfig {
	f, err := os.Open(configFile)
	if err != nil {
		panic(fmt.Sprintf("cannot open config file: %v", err))
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	config := &APIConfig{}
	err = decoder.Decode(config) //nolint:musttag // we mix yaml and env tags
	if err != nil {
		panic(fmt.Sprintf("cannot decode config file: %v", err))
	}

	ctx := context.Background()
	err = envconfig.Process(ctx, config)
	if err != nil {
		panic(fmt.Sprintf("cannot process environment variables: %v", err))
	}

	return config
}

// MustParseEnvFile loads a user defined .env file
// or the default .env file if it exists.
//
// * User defined .env file must exist and be valid.
// * Default .env file is optional.
func MustParseEnvFile(envFile *string) {
	if envFile != nil && *envFile != "" {
		if err := godotenv.Load(*envFile); err != nil {
			panic(fmt.Sprintf("Cannot load .env file: %v", err))
		}
	} else {
		_ = godotenv.Load()
	}
}

type CmdLineArgs struct {
	DebugMode  bool
	ConfigFile string
	EnvFile    string
}

func ParseCmdLineArgs() CmdLineArgs {
	var args CmdLineArgs
	flag.BoolVar(&args.DebugMode, "debug", false, "Enable debug mode")
	flag.StringVar(&args.ConfigFile, "config", "config.yml", "Config file path")
	flag.StringVar(&args.EnvFile, "env", "", ".env file path (if not set, will use .env if exists)")
	flag.Parse()

	return args
}
