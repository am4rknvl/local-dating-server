💌 Ethiopia Dating App MVP (18+)

Connect safely. Love locally.

Welcome to the Ethiopia Dating App MVP, designed for young adults and college students in Ethiopia. This interactive guide will help you set up, run, and extend the MVP.

🏗️ Features

Core MVP Flows:

Onboarding & Authentication

Email or phone (OTP optional)

Age verification (18+)

Profile setup (name, photo, bio, interests)

Profile & Discovery

View/edit profile

Browse users: swipe or scroll feed

Filters: age, gender, location, interests

Basic matching algorithm

Matching Flow

Like/dislike

Mutual like → match

Optional: favorite profiles

Messaging

1:1 chat for matched users

Text + emojis

Optional images in chat

Safety & Moderation

Block/report users

Admin moderation dashboard

Optional profile verification

Notifications

Push notifications for new matches and messages

Optional email or in-app alerts

🛠️ Tech Stack
Layer	Technology
Backend	Go (Gin / Fiber / Echo)
Database	PostgreSQL
Caching	Redis
Real-time	WebSockets (Gorilla / Fiber WS)
File Storage	AWS S3 / MinIO
Mobile App	React Native + Expo
Push Notifications	Firebase Cloud Messaging
Auth	JWT Tokens
⚡ Quick Start

Interactive steps — copy & paste to run the MVP locally.

1️⃣ Backend Setup
# Clone the repo
git clone https://github.com/yourusername/ethiopia-dating-app.git
cd ethiopia-dating-app/backend

# Install dependencies
go mod tidy

# Set environment variables
cp .env.example .env
# Update DATABASE_URL, REDIS_URL, JWT_SECRET, S3_BUCKET

# Run migrations
go run main.go migrate

# Start server
go run main.go


✅ Hint: Visit http://localhost:8080/docs for API docs (Swagger/OpenAPI).

2️⃣ Frontend Setup
cd ../mobile
npm install
expo start


Open in Expo Go on your phone or simulator

Default API URL: http://localhost:8080

3️⃣ Admin Panel
cd ../admin
npm install
npm run dev


Dashboard: http://localhost:3000

Manage users, moderation, metrics

4️⃣ Optional: OTP Integration

Use Twilio, Africa’s Talking, or a local SMS provider

Toggle OTP verification in .env:

OTP_ENABLED=false

🧩 Interactive Commands

Create a test user:

curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Amar","email":"amar@example.com","dob":"2002-05-15"}'


Like another user:

curl -X POST http://localhost:8080/matches \
  -H "Authorization: Bearer <JWT_TOKEN>" \
  -d '{"liked_user_id":2}'


Send a chat message:

wscat -c ws://localhost:8080/ws \
  -H "Authorization: Bearer <JWT_TOKEN>"

🔮 Roadmap / Next Steps

Add AI-powered matching algorithm

Premium subscription / in-app payments (Telebirr, Chapa)

Video and voice calls

Advanced moderation tools and content filters

Expand to diaspora communities

⚡ Tips for Contributors

Follow Go idioms: internal/handlers, internal/models, ws/manager.go

Use Redis for caching hot matches and session data

Keep WebSocket logic modular for future features

💡 Interactive Section

Test yourself while coding:

Can you add a “favorite” feature? ✅ Try updating the match endpoint.

Can you implement a push notification for unread messages? ✅ Use FCM SDK.

How would you scale WebSockets for 10k simultaneous users? ✅ Think Redis Pub/Sub or NATS.

📜 License

MIT © 2025 Amar
