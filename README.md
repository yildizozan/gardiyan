# Gardiyan S3-Compatible Proxy Web Server

Gardiyan, HTTP request'lerin path'ine göre S3-compatible storage'dan dosya çekip serve eden Go ile yazılmış bir proxy web server'dır.

## Özellikler

- HTTP request path'ine göre S3-compatible storage'dan dosya serve etme
- AWS S3, MinIO, Huawei OBS ve diğer S3-compatible servisler desteği
- Otomatik Content-Type belirleme
- Health check endpoint
- Configurable endpoint, region ve bucket
- Lightweight ve hızlı

## Kurulum

### Gereksinimler

- Go 1.21+
- S3-compatible storage (AWS S3, MinIO, Huawei OBS, vb.)
- Storage kimlik bilgileri

### Bağımlılıkları Yükle

```bash
go mod download
```

## Kullanım

### Environment Variables

Aşağıdaki environment variable'ları ayarlayın:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ACCESS_KEY_ID` | ✅ | - | S3-compatible storage access key |
| `SECRET_ACCESS_KEY` | ✅ | - | S3-compatible storage secret key |
| `S3_BUCKET_NAME` | ✅ | - | Target bucket name |
| `S3_ENDPOINT` | ✅ | - | S3-compatible endpoint URL |
| `REGION` | ✅ | - | Storage region |
| `S3_DISABLE_SSL` | ❌ | `false` | Disable SSL/TLS (use HTTP instead of HTTPS) |
| `S3_FORCE_PATH_STYLE` | ❌ | `false` | Use path-style URLs instead of virtual-hosted-style |
| `PORT` | ❌ | `8080` | Server listen port |

#### S3-Compatible Storage Configuration
- `S3_ENDPOINT`: S3-compatible server endpoint (örn: http://localhost:9000, obs.tr-west-1.myhuaweicloud.com)
- `S3_DISABLE_SSL`: SSL'i devre dışı bırak (true/false, varsayılan: false)
- `S3_FORCE_PATH_STYLE`: Path style kullanım (true/false, varsayılan: false)

#### Storage Credentials
- `ACCESS_KEY_ID`: Storage access key
- `SECRET_ACCESS_KEY`: Storage secret key  
- `REGION`: Storage region (varsayılan: us-east-1)

#### General Settings
- `S3_BUCKET_NAME`: Storage bucket adı (gerekli)
- `PORT`: Server port (varsayılan: 8080)

### .env Dosyası

`.env.example` dosyasını `.env` olarak kopyalayın ve değerleri düzenleyin:

```bash
cp .env.example .env
```

### Server'ı Başlatma

```bash
go run main.go
```

## API Endpoints

### Health Check
```
GET /health
```
Server'ın çalışır durumda olduğunu kontrol eder.

### Dosya Serve Etme

```
GET /{dosya_yolu}
```
S3_BUCKET_NAME environment variable'ında tanımlı bucket'tan dosyayı serve eder.

## Örnekler

### Basit dosya isteme
```bash
curl http://localhost:8080/images/logo.png
```
`S3_BUCKET_NAME` environment variable'ında tanımlı bucket'taki `images/logo.png` dosyasını döner.

### JSON dosyası isteme
```bash
curl http://localhost:8080/config/settings.json
```
`S3_BUCKET_NAME` environment variable'ında tanımlı bucket'taki `config/settings.json` dosyasını döner.

## Configuration Examples

### AWS S3
```bash
# .env file
S3_ENDPOINT=
S3_BUCKET_NAME=my-aws-bucket
ACCESS_KEY_ID=AKIA...
SECRET_ACCESS_KEY=...
REGION=us-west-2
```

### MinIO
```bash
# .env file
S3_ENDPOINT=http://localhost:9000
S3_DISABLE_SSL=true
S3_FORCE_PATH_STYLE=true
S3_BUCKET_NAME=my-minio-bucket
ACCESS_KEY_ID=minioadmin
SECRET_ACCESS_KEY=minioadmin
```

### Huawei Cloud OBS
```bash
# .env file
S3_ENDPOINT=obs.tr-west-1.myhuaweicloud.com
S3_DISABLE_SSL=false
S3_FORCE_PATH_STYLE=false
S3_BUCKET_NAME=my-storage-bucket
ACCESS_KEY_ID=...
SECRET_ACCESS_KEY=...
REGION=tr-west-1
```

## Güvenlik Notları

- Storage kimlik bilgilerini güvenli şekilde saklayın
- Storage bucket izinlerini kontrol edin
- Production ortamında HTTPS kullanın
- Rate limiting ve authentication ekleyebilirsiniz

## Lisans

MIT License
