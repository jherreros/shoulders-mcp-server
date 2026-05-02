package config

type ProfileSpec struct {
	Name                string
	EventStreams        bool
	PolicyReporter      bool
	Trivy               bool
	Falco               bool
	HubbleUI            bool
	LogTracePipeline    bool
	CiliumObservability bool
	PostgresInstances   int
	KafkaReplicas       int
	KafkaMinISR         int
	KafkaStorage        string
	PrometheusRetention string
	PrometheusSize      string
}

func (cfg *Config) ProfileSpec() ProfileSpec {
	return ProfileSpecFor(cfg.Profile())
}

func ProfileSpecFor(profile string) ProfileSpec {
	switch normalizeProfile(profile) {
	case ProfileSmall:
		return ProfileSpec{
			Name:                ProfileSmall,
			PostgresInstances:   1,
			KafkaReplicas:       1,
			KafkaMinISR:         1,
			KafkaStorage:        "1Gi",
			PrometheusRetention: "6h",
			PrometheusSize:      "512MiB",
		}
	case ProfileLarge:
		return ProfileSpec{
			Name:                ProfileLarge,
			EventStreams:        true,
			PolicyReporter:      true,
			Trivy:               true,
			Falco:               true,
			HubbleUI:            true,
			LogTracePipeline:    true,
			CiliumObservability: true,
			PostgresInstances:   2,
			KafkaReplicas:       3,
			KafkaMinISR:         2,
			KafkaStorage:        "1Gi",
			PrometheusRetention: "7d",
			PrometheusSize:      "5GiB",
		}
	default:
		return ProfileSpec{
			Name:                ProfileMedium,
			EventStreams:        true,
			PolicyReporter:      true,
			Trivy:               true,
			Falco:               true,
			HubbleUI:            true,
			LogTracePipeline:    true,
			CiliumObservability: true,
			PostgresInstances:   2,
			KafkaReplicas:       3,
			KafkaMinISR:         2,
			KafkaStorage:        "1Gi",
			PrometheusRetention: "24h",
			PrometheusSize:      "1GiB",
		}
	}
}
