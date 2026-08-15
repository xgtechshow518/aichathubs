# AIChatsHub

A modern admin portal for managing customer conversations across multiple social media platforms. Built with React (frontend) and Go (backend).

## Features

- **Social Login**: Google and Facebook OAuth 2.0 authentication
- **Dashboard**: Overview of chat statistics and recent conversations
- **Chat History**: Real-time chat interface with WebSocket support
- **Multi-platform**: Support for WhatsApp, Facebook Messenger, and more

## Tech Stack

### Frontend
- React 18 with TypeScript
- Ant Design for UI components
- React Router for navigation
- Zustand for state management
- Axios for HTTP requests
- WebSocket for real-time updates

### Backend
- Go with Echo framework
- PostgreSQL with GORM
- JWT for authentication
- Gorilla WebSocket for real-time communication

## Project Structure

```
smart-live-chats/
├── frontend/                 # React application
│   ├── src/
│   │   ├── components/       # Reusable UI components
│   │   ├── pages/            # Page components
│   │   ├── services/         # API and WebSocket clients
│   │   ├── store/            # Zustand stores
│   │   ├── hooks/            # Custom hooks
│   │   └── types/            # TypeScript types
│   └── package.json
├── backend/                  # Go application
│   ├── cmd/server/           # Main entry point
│   ├── internal/
│   │   ├── handlers/         # HTTP and WebSocket handlers
│   │   ├── models/           # Database models
│   │   ├── services/         # Business logic
│   │   ├── middleware/       # Auth, CORS middleware
│   │   ├── database/         # DB connection
│   │   └── config/           # Configuration
│   └── go.mod
└── README.md
```

## Getting Started

### Prerequisites

- Node.js 18+ and npm
- Go 1.21+
- PostgreSQL 14+

### Database Setup

1. Create a PostgreSQL database:

```sql
CREATE DATABASE smart_live_chats;
```

### Backend Setup

1. Navigate to the backend directory:

```bash
cd backend
```

2. Copy the environment file and configure it:

```bash
cp env.example .env
```

3. Update the `.env` file with your settings:

```env
DATABASE_URL=postgres://user:password@localhost:5432/smart_live_chats?sslmode=disable
JWT_SECRET=your-secret-key
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
FACEBOOK_CLIENT_ID=your-facebook-app-id
FACEBOOK_CLIENT_SECRET=your-facebook-app-secret
```

4. Download dependencies and run:

```bash
go mod tidy
go run cmd/server/main.go
```

The backend will start on `http://localhost:8080`.

### Frontend Setup

1. Navigate to the frontend directory:

```bash
cd frontend
```

2. Install dependencies:

```bash
npm install
```

3. Copy the environment file and configure it:

```bash
cp env.example .env
```

4. Update the `.env` file:

```env
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080
VITE_GOOGLE_CLIENT_ID=your-google-client-id
VITE_FACEBOOK_APP_ID=your-facebook-app-id
```

5. Start the development server:

```bash
npm run dev
```

The frontend will start on `http://localhost:5173`.

## OAuth Setup

### Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Enable Google+ API
4. Go to Credentials → Create Credentials → OAuth 2.0 Client IDs
5. Add authorized redirect URIs:
   - `http://localhost:5173/login` (development)
   - `http://localhost:8080/api/auth/google/callback` (backend)

### Facebook OAuth

1. Go to [Facebook Developers](https://developers.facebook.com/)
2. Create a new app or select existing
3. Add Facebook Login product
4. Configure OAuth redirect URIs:
   - `http://localhost:5173/login` (development)
   - `http://localhost:8080/api/auth/facebook/callback` (backend)

## API Endpoints

### Authentication
- `GET /api/auth/google` - Get Google OAuth URL
- `POST /api/auth/google/callback` - Google OAuth callback
- `GET /api/auth/facebook` - Get Facebook OAuth URL
- `POST /api/auth/facebook/callback` - Facebook OAuth callback
- `GET /api/auth/me` - Get current user (requires auth)

### Chats
- `GET /api/chats` - List chat sessions
- `GET /api/chats/stats` - Get chat statistics
- `GET /api/chats/:id` - Get single chat session
- `GET /api/chats/:id/messages` - Get chat messages
- `POST /api/chats/:id/messages` - Send a message
- `POST /api/chats/:id/read` - Mark messages as read

### WebSocket
- `GET /ws` - WebSocket connection for real-time updates

## Development

### Running Tests

```bash
# Backend tests
cd backend
go test ./...

# Frontend tests
cd frontend
npm test
```

### Building for Production

```bash
# Backend
cd backend
go build -o server cmd/server/main.go

# Frontend
cd frontend
npm run build
```

## License

MIT

