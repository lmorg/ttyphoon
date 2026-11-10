package config

type S3Config struct {
	Endpoint        string `yaml:"Endpoint"`
	Region          string `yaml:"Region"`
	AccessKeyID     string `yaml:"AccessKeyID"`
	SecretAccessKey string `yaml:"SecretAccessKey"`
	SessionToken    string `yaml:"SessionToken"`
	Bucket          string `yaml:"Bucket"`
	Prefix          string `yaml:"Prefix"`
	PublicBaseURL   string `yaml:"PublicBaseURL"`
}
