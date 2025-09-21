# Ethiopia Dating App MVP

A comprehensive dating app MVP built for Ethiopia with Go backend, PostgreSQL database, Redis caching, and React Native frontend.

## Features

### Core Features
- **User Authentication**: Email/phone registration with optional OTP verification
- **Age Verification**: 18+ requirement with date of birth validation
- **Profile Management**: Photo uploads, bio, interests, location
- **Discovery**: Swipe-style or scroll feed with advanced filtering
- **Matching System**: Mutual likes create matches
- **Real-time Messaging**: WebSocket-based chat for matched users
- **Safety Features**: Block, report, and admin moderation
- **Push Notifications**: Firebase integration for real-time alerts

### Technical Features
- **Backend**: Go with Gin framework
- **Database**: PostgreSQL with GORM ORM
- **Caching**: Redis for sessions and hot data
- **File Storage**: AWS S3 or MinIO for profile photos
- **Real-time**: WebSocket support for messaging
- **Authentication**: JWT tokens with refresh mechanism
- **Admin Panel**: User management and analytics

## Tech Stack

### Backend
- **Language**: Go 1.21+
- **Framework**: Gin
- **Database**: PostgreSQL 15+
- **ORM**: GORM
- **Cache**: Redis 7+
- **Storage**: AWS S3 / MinIO
- **WebSocket**: Gorilla WebSocket
- **Authentication**: JWT

### Frontend (React Native)
- **Framework**: React Native with Expo
- **State Management**: Redux Toolkit
- **Navigation**: React Navigation
- **UI**: NativeBase or React Native Elements
- **Push Notifications**: Firebase Cloud Messaging

## Quick Start

### Prerequisites
- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Node.js 18+ (for React Native)
- Docker & Docker Compose (optional)

### Using Docker Compose (Recommended)

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd ethiopia-dating-app
   ```

2. **Start services**
   ```bash
   docker-compose up -d
   ```

3. **Initialize database**
   ```bash
   # The database will be automatically migrated when the app starts
   ```

4. **Access services**
   - API: http://localhost:8080
   - MinIO Console: http://localhost:9001 (admin/admin)
   - PostgreSQL: localhost:5432
   - Redis: localhost:6379

### Manual Setup

1. **Install dependencies**
   ```bash
   go mod download
   ```

2. **Setup environment**
   ```bash
   cp env.example .env
   # Edit .env with your configuration
   ```

3. **Start PostgreSQL and Redis**
   ```bash
   # Using Docker
   docker run -d --name postgres -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:15
   docker run -d --name redis -p 6379:6379 redis:7-alpine
   ```

4. **Run the application**
   ```bash
   go run main.go
   ```

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/verify-otp` - Verify OTP
- `POST /api/v1/auth/refresh` - Refresh token
- `POST /api/v1/auth/logout` - User logout

### User Management
- `GET /api/v1/users/profile` - Get user profile
- `PUT /api/v1/users/profile` - Update profile
- `POST /api/v1/users/profile/photo` - Upload photo
- `DELETE /api/v1/users/profile/photo/:id` - Delete photo
- `GET /api/v1/users/discover` - Discover users
- `GET /api/v1/users/favorites` - Get favorites
- `POST /api/v1/users/favorites/:user_id` - Add to favorites
- `DELETE /api/v1/users/favorites/:user_id` - Remove from favorites

### Matching
- `POST /api/v1/matches/like/:user_id` - Like user
- `POST /api/v1/matches/dislike/:user_id` - Dislike user
- `GET /api/v1/matches` - Get matches
- `DELETE /api/v1/matches/:match_id` - Unmatch

### Messaging
- `GET /api/v1/messages/conversations` - Get conversations
- `GET /api/v1/messages/conversations/:id` - Get messages
- `POST /api/v1/messages/conversations/:id` - Send message
- `PUT /api/v1/messages/conversations/:id/read` - Mark as read
- `GET /api/v1/ws` - WebSocket connection

### Safety
- `POST /api/v1/users/block/:user_id` - Block user
- `DELETE /api/v1/users/block/:user_id` - Unblock user
- `POST /api/v1/users/report` - Report user

### Admin
- `GET /api/v1/admin/users` - Get all users
- `GET /api/v1/admin/users/:id` - Get user details
- `PUT /api/v1/admin/users/:id/status` - Update user status
- `GET /api/v1/admin/reports` - Get reports
- `PUT /api/v1/admin/reports/:id/status` - Update report status
- `GET /api/v1/admin/analytics` - Get analytics

## Database Schema

### Core Tables
- `users` - User profiles and authentication
- `profile_photos` - User profile pictures
- `interests` - Available interests/categories
- `user_interests` - User-interest relationships
- `matches` - Mutual likes between users
- `conversations` - Chat conversations
- `messages` - Individual messages
- `reports` - User reports and moderation
- `blocked_users` - Blocked user relationships

### Authentication Tables
- `otps` - OTP verification codes
- `user_sessions` - Active user sessions
- `notifications` - Push notifications

### Admin Tables
- `admins` - Admin users
- `user_activities` - User activity logs

## Configuration

### Environment Variables

```bash
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/ethiopia_dating_app?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# JWT
JWT_SECRET=your-super-secret-jwt-key-here
JWT_EXPIRY=24h

# Server
PORT=8080
GIN_MODE=debug

# Storage (AWS S3 or MinIO)
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
AWS_REGION=us-east-1
S3_BUCKET=ethiopia-dating-photos

# MinIO (alternative to S3)
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false

# Firebase (for push notifications)
FIREBASE_PROJECT_ID=your-firebase-project-id
FIREBASE_PRIVATE_KEY_PATH=./firebase-private-key.json

# OTP (optional)
OTP_ENABLED=true
OTP_EXPIRY=5m

# File upload limits
MAX_FILE_SIZE=10485760
ALLOWED_IMAGE_TYPES=image/jpeg,image/png,image/webp
```

## Development

### Project Structure
```
ethiopia-dating-app/
├── main.go                 # Application entry point
├── go.mod                  # Go dependencies
├── docker-compose.yml      # Docker services
├── Dockerfile             # Container configuration
├── internal/              # Internal packages
│   ├── config/           # Configuration management
│   ├── database/         # Database setup and migrations
│   ├── handlers/         # HTTP request handlers
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Database models
│   ├── redis/            # Redis client
│   ├── services/         # Business logic services
│   ├── utils/            # Utility functions
│   └── websocket/        # WebSocket handling
└── mobile/               # React Native app (to be created)
```

### Running Tests
```bash
go test ./...
```

### Database Migrations
Migrations are handled automatically by GORM when the application starts.

### Adding New Features
1. Create models in `internal/models/`
2. Add handlers in `internal/handlers/`
3. Update routes in `main.go`
4. Add middleware if needed in `internal/middleware/`

## Security Considerations

- **Age Verification**: Strict 18+ requirement
- **Input Validation**: All inputs are validated
- **SQL Injection**: Protected by GORM
- **XSS Protection**: Input sanitization
- **Rate Limiting**: Implemented for sensitive endpoints
- **JWT Security**: Secure token generation and validation
- **File Upload**: Type and size validation
- **CORS**: Configured for cross-origin requests

## Deployment

### Production Checklist
- [ ] Set strong JWT secret
- [ ] Configure production database
- [ ] Set up Redis cluster
- [ ] Configure S3/MinIO with proper permissions
- [ ] Set up Firebase for push notifications
- [ ] Configure CORS for production domains
- [ ] Set up SSL/TLS certificates
- [ ] Configure logging and monitoring
- [ ] Set up backup strategies
- [ ] Configure rate limiting

### Environment Setup
1. **Database**: Use managed PostgreSQL service
2. **Redis**: Use managed Redis service
3. **Storage**: Use AWS S3 or managed MinIO
4. **Monitoring**: Set up application monitoring
5. **Logging**: Configure structured logging

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions, please open an issue in the repository.

## Roadmap

### Phase 1 (MVP) - Current
- [x] User authentication and registration
- [x] Profile management
- [x] Discovery and matching
- [x] Real-time messaging
- [x] Safety features
- [x] Admin panel

### Phase 2 (Future)
- [ ] Advanced matching algorithms
- [ ] Video calling
- [ ] Premium features
- [ ] Advanced analytics
- [ ] Mobile app optimization
- [ ] Multi-language support

### Phase 3 (Scale)
- [ ] Microservices architecture
- [ ] Advanced AI features
- [ ] International expansion
- [ ] Advanced security features
