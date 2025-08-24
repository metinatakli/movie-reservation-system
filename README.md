# Movie Reservation System

[![Go Version](https://img.shields.io/badge/Go-1.24.0-blue.svg)](https://golang.org/)

My learning-by-doing project.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Configuration](#configuration)
- [Installation & Setup](#installation--setup)
- [Monitoring & Observability](#monitoring--observability)

## Overview

The movie reservation system is a backend application that enables users to browse movies, view showtimes, reserve seats, and complete payments. The system includes user registration with email activation, secure authentication, cart management, Stripe payment integration, and monitoring with OpenTelemetry, Prometheus, Grafana, and Jaeger.

### Technology Stack

- **Backend**: Go 1.24.0
- **Database**: PostgreSQL with PostGIS
- **Cache**: Redis
- **Payments**: Stripe integration
- **Monitoring**: OpenTelemetry, Prometheus, Grafana, Jaeger, Loki
- **Containerization**: Docker & Docker Compose
- **Testing**: Testcontainers for integration tests

## Prerequisites

- Go 1.24.0 or higher
- Docker and Docker Compose
- PostgreSQL 15+ (if running locally)
- Redis 7+ (if running locally)
- Make (optional, for using Makefile commands)

## Configuration

### Environment Variables

Create a `.env` file with the following variables:

```env
# Database
DB_DSN=postgres://username:password@localhost:5432/movie_reservation?sslmode=disable
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password
POSTGRES_DB=movie_reservation

# Redis
REDIS_URL=redis://localhost:6379

# SMTP
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587

# Stripe
STRIPE_KEY=sk_test_your_stripe_secret_key
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_secret

# OpenTelemetry
OTEL_COLLECTOR_URL=http://localhost:4317

# Application
PORT=3000
ENVIRONMENT=development
```

## Installation & Setup

### Quick Start with Docker

1. **Clone the repository**
   ```bash
   git clone https://github.com/metinatakli/movie-reservation-system.git
   cd movie-reservation-system
   ```

2. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Start the application**
   ```bash
   make docker/up
   ```

The application will be available at http://localhost:3000

### Manual Installation

1. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

2. **Ensure PostgreSQL and Redis are running**
   Make sure PostgreSQL and Redis are available at the URLs specified in your `.env` file.

3. **Install dependencies**
   ```bash
   go mod download
   ```

4. **Generate API code**
   ```bash
   make generate
   ```

5. **Set up database**
   ```bash
   make db/migrations/up
   ```

6. **Run the application**
   ```bash
   make run
   ```

### External Services Setup

1. **Stripe**: Create a Stripe account and get API keys. It's not necessary for the application to run, but it's necessary if you want to try a payment process.
2. **SMTP**: Configure email service (Gmail, SendGrid, etc.). I used mailtrap as it's free.

## Monitoring & Observability

The system includes monitoring and observability tools for tracking application performance, debugging issues, and understanding system behavior. These tools help you monitor metrics, trace requests, and analyze logs.

### Accessing Monitoring Tools

- **Grafana**: http://localhost:3001 (admin/admin)
- **Jaeger**: http://localhost:16686
- **Prometheus**: http://localhost:9090
- **Loki**: http://localhost:3100
