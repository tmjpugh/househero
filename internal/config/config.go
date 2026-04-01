package config

import "os"

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	JWTSecret  string
	Port       string

	// MQTT settings (all optional; MQTT is disabled when MQTTBroker is empty)
	MQTTBroker   string // e.g. "tcp://mosquitto:1883"
	MQTTClientID string
	MQTTUsername string
	MQTTPassword string
}

func Load() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "househero"),
		DBPassword: getEnv("DB_PASSWORD", "househero_dev"),
		DBName:     getEnv("DB_NAME", "househero_db"),
		JWTSecret:  getEnv("JWT_SECRET", "your-secret-key"),
		Port:       getEnv("PORT", "8080"),

		MQTTBroker:   getEnv("MQTT_BROKER", ""),
		MQTTClientID: getEnv("MQTT_CLIENT_ID", "househero"),
		MQTTUsername: getEnv("MQTT_USERNAME", ""),
		MQTTPassword: getEnv("MQTT_PASSWORD", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
