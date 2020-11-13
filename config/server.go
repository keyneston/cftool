package config

type ServerCacheEntry struct {
	PrivateIP  string `yaml:"private_ip" json:"private_ip"`
	PublicIP   string `yaml:"public_ip" json:"public_ip"`
	ARN        string `yaml:"arn" json:"arn"`
	PrivateDNS string `yaml:"private_dns" json:"private_dns"`
	PublicDNS  string `yaml:"public_dns" json:"public_dns"`
	VPCID      string `yaml:"vpc_id" json:"vpc_id"`
}
