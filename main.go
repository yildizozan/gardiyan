package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

var logLevel string

type GardiyanServer struct {
	s3Client   *s3.S3
	bucketName string
	port       string
}

func NewGardiyanServer() *GardiyanServer {
	// S3-compatible storage endpoint and region configuration
	endpoint := getEnvOrDefault("S3_ENDPOINT", "")
	region := getEnvOrDefault("REGION", getEnvOrDefault("AWS_REGION", "us-east-1"))
	
	config := &aws.Config{
		Region: aws.String(region),
	}
	
	// If S3 endpoint is specified, use it (for MinIO, Huawei OBS, etc.)
	if endpoint != "" {
		config.Endpoint = aws.String(endpoint)
		// Path style vs Virtual host style configuration
		// false = Virtual host style (bucket.endpoint.com)
		// true = Path style (endpoint.com/bucket)
		forcePathStyle := getEnvOrDefault("S3_FORCE_PATH_STYLE", "false") == "true"
		config.S3ForcePathStyle = aws.Bool(forcePathStyle)
		config.DisableSSL = aws.Bool(getEnvOrDefault("S3_DISABLE_SSL", "false") == "true")
		log.Printf("Using S3-compatible endpoint: %s (PathStyle: %v)", endpoint, forcePathStyle)
	} else {
		log.Printf("Using AWS S3 (region: %s)", region)
	}

	// Create S3-compatible session
	sess, err := session.NewSession(config)
	if err != nil {
		log.Fatal("Failed to create S3 session:", err)
	}

	return &GardiyanServer{
		s3Client:   s3.New(sess),
		bucketName: getEnvOrDefault("S3_BUCKET_NAME", ""),
		port:       getEnvOrDefault("PORT", "8080"),
	}
}

func (gs *GardiyanServer) proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Get request path
	path := r.URL.Path
	
	// Clean root path
	if strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}
	
	// Check for empty path
	if path == "" {
		http.Error(w, "File path not specified", http.StatusBadRequest)
		return
	}

	// For this implementation, we always use the default bucket from S3_BUCKET_NAME
	// The path from HTTP request becomes the S3 object key directly
	bucketName := gs.bucketName
	objectKey := path
	
	if bucketName == "" {
		http.Error(w, "S3_BUCKET_NAME environment variable not defined", http.StatusBadRequest)
		return
	}

	// Get S3-compatible endpoint to construct the full S3 URL
	endpoint := getEnvOrDefault("S3_ENDPOINT", "")
	var fullS3URL string
	if endpoint != "" {
		// Remove protocol from endpoint if present
		cleanEndpoint := strings.TrimPrefix(endpoint, "http://")
		cleanEndpoint = strings.TrimPrefix(cleanEndpoint, "https://")
		
		// Construct full S3 URL: https://bucket.endpoint/path
		protocol := "https"
		if getEnvOrDefault("S3_DISABLE_SSL", "false") == "true" {
			protocol = "http"
		}
		fullS3URL = fmt.Sprintf("%s://%s.%s/%s", protocol, bucketName, cleanEndpoint, objectKey)
	} else {
		// AWS S3 format
		region := getEnvOrDefault("AWS_REGION", "us-east-1")
		fullS3URL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, region, objectKey)
	}

	debugLog("S3 URL: %s", fullS3URL)
	debugLog("Guard checking storage cell for prisoner: %s in facility: %s", objectKey, bucketName)

	// Get file from S3-compatible storage using the object key
	result, err := gs.s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		clientIP := r.RemoteAddr
		if forwardedIP := r.Header.Get("X-Forwarded-For"); forwardedIP != "" {
			clientIP = forwardedIP
		}
		
		// Debug level: detailed S3 error information
		debugLog("S3 file retrieval error: %v", err)
		debugLog("Failed S3 URL would be: %s", fullS3URL)
		
		// Check for specific AWS errors
		if awsErr, ok := err.(awserr.Error); ok {
			switch awsErr.Code() {
			case "AccessDenied":
				log.Printf("AccessDenied - %s - %s", clientIP, fullS3URL)
				http.Error(w, "🔐 Access Denied! The guard's security clearance is insufficient to enter this restricted wing of the prison! 👮‍♂️🚫", http.StatusForbidden)
				return
			case "NoSuchKey":
				log.Printf("NoSuchKey - %s - %s", clientIP, fullS3URL)
				http.Error(w, "🚫 Sorry! That prisoner has escaped from the cell. The warden is still searching the facility! 👮‍♂️", http.StatusNotFound)
				return
			case "NoSuchBucket":
				log.Printf("NoSuchBucket - %s - %s", clientIP, fullS3URL)
				http.Error(w, "🏢 Prison facility not found! The entire wing seems to have vanished from the records! 📋❌", http.StatusNotFound)
				return
			}
		}
		
		// Check for generic AccessDenied in error message (fallback)
		if strings.Contains(err.Error(), "AccessDenied") {
			log.Printf("GenericAccessDenied - %s - %s", clientIP, fullS3URL)
			http.Error(w, "🔐 Access Denied! The guard's security clearance is insufficient to enter this restricted wing of the prison! 👮‍♂️🚫", http.StatusForbidden)
			return
		}
		
		// Default error message
		log.Printf("UnknownError - %s - %s", clientIP, fullS3URL)
		http.Error(w, "🚫 Sorry! That prisoner has escaped from the cell. The warden is still searching the facility! 👮‍♂️", http.StatusNotFound)
		return
	}
	defer result.Body.Close()

	// Determine Content-Type
	contentType := getContentType(objectKey)
	w.Header().Set("Content-Type", contentType)
	
	// Set Content-Length
	if result.ContentLength != nil {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", *result.ContentLength))
	}

	// Copy file to response
	_, err = io.Copy(w, result.Body)
	if err != nil {
		log.Printf("Response write error: %v", err)
		return
	}

	// Get client IP for successful request logging
	clientIP := r.RemoteAddr
	if forwardedIP := r.Header.Get("X-Forwarded-For"); forwardedIP != "" {
		clientIP = forwardedIP
	}
	
	log.Printf("Success - %s - %s", clientIP, fullS3URL)
}

func (gs *GardiyanServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("🔒 Gardiyan is on duty and the prison is secure! 👮‍♂️"))
}

func (gs *GardiyanServer) Start() {
	router := mux.NewRouter()
	
	// Health check endpoint
	router.HandleFunc("/health", gs.healthCheck).Methods("GET")
	
	// Main proxy handler - for all other paths
	router.PathPrefix("/").HandlerFunc(gs.proxyHandler).Methods("GET")

	log.Printf("🔒 Gardiyan is starting shift - Guard post: %s", gs.port)
	log.Printf("📋 Prison facility: %s", gs.bucketName)
	
	// Show the S3 URL format that will be used
	endpoint := getEnvOrDefault("S3_ENDPOINT", "")
	if endpoint != "" {
		cleanEndpoint := strings.TrimPrefix(endpoint, "http://")
		cleanEndpoint = strings.TrimPrefix(cleanEndpoint, "https://")
		protocol := "https"
		if getEnvOrDefault("S3_DISABLE_SSL", "false") == "true" {
			protocol = "http"
		}
		log.Printf("🏢 Storage facility format: %s://%s.%s/{prisoner}", protocol, gs.bucketName, cleanEndpoint)
	} else {
		region := getEnvOrDefault("AWS_REGION", "us-east-1")
		log.Printf("🏢 Storage facility format: https://%s.s3.%s.amazonaws.com/{prisoner}", gs.bucketName, region)
	}
	
	log.Printf("📝 Visiting example:")
	log.Printf("  👤 Visitor request: http://localhost:%s/images/logo.png", gs.port)
	if endpoint != "" {
		cleanEndpoint := strings.TrimPrefix(endpoint, "http://")
		cleanEndpoint = strings.TrimPrefix(cleanEndpoint, "https://")
		protocol := "https"
		if getEnvOrDefault("S3_DISABLE_SSL", "false") == "true" {
			protocol = "http"
		}
		log.Printf("  🔓 Released from: %s://%s.%s/images/logo.png", protocol, gs.bucketName, cleanEndpoint)
	}
	
	if err := http.ListenAndServe(":"+gs.port, router); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func debugLog(format string, v ...interface{}) {
	if logLevel == "debug" {
		log.Printf("[DEBUG] "+format, v...)
	}
}

func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".xml":
		return "application/xml"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

func validateEnvironmentVariables() {
	var missingVars []string
	
	// Check required S3 credentials
	if getEnvOrDefault("ACCESS_KEY_ID", "") == "" {
		missingVars = append(missingVars, "ACCESS_KEY_ID")
	}
	
	if getEnvOrDefault("SECRET_ACCESS_KEY", "") == "" {
		missingVars = append(missingVars, "SECRET_ACCESS_KEY")
	}
	
	// Check required S3 bucket name
	if getEnvOrDefault("S3_BUCKET_NAME", "") == "" {
		missingVars = append(missingVars, "S3_BUCKET_NAME")
	}
	
	// Check required S3 endpoint (no default for MinIO/custom S3)
	if getEnvOrDefault("S3_ENDPOINT", "") == "" {
		missingVars = append(missingVars, "S3_ENDPOINT")
	}
	
	// Check required region
	if getEnvOrDefault("REGION", "") == "" {
		missingVars = append(missingVars, "REGION")
	}
	
	// If any required variables are missing, exit with error
	if len(missingVars) > 0 {
		log.Printf("🚨 CRITICAL: Missing required environment variables!")
		log.Printf("🔧 The following environment variables must be set:")
		for _, varName := range missingVars {
			log.Printf("   ❌ %s", varName)
		}
		log.Printf("")
		log.Printf("💡 Complete .env file example:")
		log.Printf("   ACCESS_KEY_ID=your-access-key")
		log.Printf("   SECRET_ACCESS_KEY=your-secret-key")
		log.Printf("   S3_BUCKET_NAME=your-bucket-name")
		log.Printf("   S3_ENDPOINT=your-s3-endpoint")
		log.Printf("   REGION=your-region")
		log.Printf("   S3_FORCE_PATH_STYLE=false")
		log.Printf("   S3_DISABLE_SSL=false  # Optional, defaults to false")
		log.Printf("   PORT=8080             # Optional, defaults to 8080")
		log.Printf("")
		log.Printf("🚫 Gardiyan cannot start without complete configuration!")
		log.Fatal("Environment validation failed - exiting")
	}
	
	log.Printf("✅ All required environment variables validated successfully")
}

func main() {
	// Parse command line flags
	flag.StringVar(&logLevel, "log-level", "info", "Log level (info, debug)")
	flag.Parse()

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, environment variables will be read from system")
	}

	debugLog("Log level set to: %s", logLevel)

	// Validate required environment variables
	validateEnvironmentVariables()

	// Set AWS SDK compatible environment variables for backward compatibility
	if accessKey := getEnvOrDefault("ACCESS_KEY_ID", ""); accessKey != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", accessKey)
	}
	if secretKey := getEnvOrDefault("SECRET_ACCESS_KEY", ""); secretKey != "" {
		os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
	}

	server := NewGardiyanServer()
	server.Start()
}
