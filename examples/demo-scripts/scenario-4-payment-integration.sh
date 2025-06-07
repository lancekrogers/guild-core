#!/bin/bash
# Scenario 4: Payment Integration with PaymentSentinel
# Duration: 3-4 minutes
# Purpose: Show security-focused payment integration

echo "🏰 Guild Framework Demo - Scenario 4: Payment Integration"
echo "========================================================"
echo ""
echo "💳 Working with PaymentSentinel on Stripe integration..."
echo ""

echo "$ guild chat --campaign e-commerce"
echo ""
echo "You: @payment-sentinel Integrate Stripe payment processing with webhook handling and PCI compliance"
echo ""
sleep 1

cat << 'EOF'
🤔 payment-sentinel is thinking...
   Analyzing payment security requirements...

PaymentSentinel: I'll implement a secure Stripe integration with webhook handling and PCI compliance. Security is my top priority.

⚙️ payment-sentinel is working...
   Implementing secure payment flow...
   [████░░░░░░░░░░░░] 25%

## Secure Payment Processing Implementation

### 1. Payment Service Structure

```go
// payment-service/internal/stripe/client.go
package stripe

import (
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    
    "github.com/stripe/stripe-go/v74"
    "github.com/stripe/stripe-go/v74/paymentintent"
    "github.com/stripe/stripe-go/v74/webhook"
)

// SecureStripeClient wraps Stripe API with additional security
type SecureStripeClient struct {
    apiKey          string
    webhookSecret   string
    encryptionKey   []byte
}

func NewSecureStripeClient(apiKey, webhookSecret string, encryptionKey []byte) *SecureStripeClient {
    stripe.Key = apiKey
    return &SecureStripeClient{
        apiKey:        apiKey,
        webhookSecret: webhookSecret,
        encryptionKey: encryptionKey,
    }
}
```

⚙️ payment-sentinel is working...
   Creating PCI-compliant token handling...
   [████████░░░░░░░░] 50%

### 2. PCI-Compliant Payment Flow

```go
// payment-service/api/handlers/payment.go
package handlers

// CreatePaymentIntent - Never store card details, use Stripe tokens
func (h *PaymentHandler) CreatePaymentIntent(c *gin.Context) {
    var req CreatePaymentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }
    
    // Validate amount and currency
    if err := h.validatePaymentAmount(req.Amount, req.Currency); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Create idempotency key for safe retries
    idempotencyKey := h.generateIdempotencyKey(req.OrderID)
    
    // Create payment intent with Stripe
    params := &stripe.PaymentIntentParams{
        Amount:   stripe.Int64(req.Amount),
        Currency: stripe.String(req.Currency),
        Metadata: map[string]string{
            "order_id":    req.OrderID,
            "customer_id": req.CustomerID,
        },
        AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
            Enabled: stripe.Bool(true),
        },
    }
    params.SetIdempotencyKey(idempotencyKey)
    
    pi, err := paymentintent.New(params)
    if err != nil {
        h.logger.Error("Payment intent creation failed", "error", err)
        c.JSON(500, gin.H{"error": "Payment processing failed"})
        return
    }
    
    // Log payment attempt (never log sensitive data)
    h.auditLogger.Log(AuditEvent{
        Type:       "payment_intent_created",
        OrderID:    req.OrderID,
        CustomerID: req.CustomerID,
        Amount:     req.Amount,
        // Never log: card numbers, CVV, full client secret
    })
    
    c.JSON(200, gin.H{
        "payment_intent_id": pi.ID,
        "client_secret":     pi.ClientSecret,
        "status":           pi.Status,
    })
}
```

⚙️ payment-sentinel is working...
   Implementing webhook security...
   [████████████░░░░] 75%

### 3. Secure Webhook Handler

```go
// payment-service/api/handlers/webhook.go
package handlers

func (h *WebhookHandler) HandleStripeWebhook(c *gin.Context) {
    payload, err := ioutil.ReadAll(c.Request.Body)
    if err != nil {
        c.JSON(400, gin.H{"error": "Invalid payload"})
        return
    }
    
    // Verify webhook signature
    signatureHeader := c.GetHeader("Stripe-Signature")
    event, err := webhook.ConstructEvent(payload, signatureHeader, h.webhookSecret)
    if err != nil {
        h.logger.Warn("Webhook signature verification failed", "error", err)
        c.JSON(400, gin.H{"error": "Invalid signature"})
        return
    }
    
    // Process webhook asynchronously to avoid timeouts
    go h.processWebhookEvent(event)
    
    // Acknowledge receipt immediately
    c.JSON(200, gin.H{"received": true})
}

func (h *WebhookHandler) processWebhookEvent(event stripe.Event) {
    ctx := context.Background()
    
    // Implement idempotency - prevent duplicate processing
    if h.isEventProcessed(ctx, event.ID) {
        h.logger.Info("Skipping duplicate event", "event_id", event.ID)
        return
    }
    
    switch event.Type {
    case "payment_intent.succeeded":
        var pi stripe.PaymentIntent
        if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
            h.logger.Error("Failed to parse payment intent", "error", err)
            return
        }
        h.handlePaymentSuccess(ctx, &pi)
        
    case "payment_intent.payment_failed":
        var pi stripe.PaymentIntent
        if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
            h.logger.Error("Failed to parse payment intent", "error", err)
            return
        }
        h.handlePaymentFailure(ctx, &pi)
    }
    
    // Mark event as processed
    h.markEventProcessed(ctx, event.ID)
}
```

### 4. Security Configuration

```yaml
# payment-service/config/security.yaml
security:
  # PCI DSS Requirements
  pci_compliance:
    # Never store sensitive cardholder data
    disable_card_storage: true
    # Use tokenization for all card operations
    require_tokenization: true
    # Enable TLS 1.2+ only
    min_tls_version: "1.2"
    
  # Rate limiting for payment endpoints
  rate_limits:
    create_payment: "10/minute"
    webhook: "1000/minute"
    
  # Encryption for sensitive data at rest
  encryption:
    algorithm: "AES-256-GCM"
    key_rotation_days: 90
```

### 5. Environment Configuration

```bash
# .env.example (never commit actual values)
STRIPE_API_KEY=sk_test_... # Use restricted keys in production
STRIPE_WEBHOOK_SECRET=whsec_...
ENCRYPTION_KEY=base64_encoded_32_byte_key
DATABASE_ENCRYPTION=enabled
AUDIT_LOG_PATH=/secure/logs/payment-audit.log
```

### 6. Security Monitoring

```go
// payment-service/internal/security/monitor.go
package security

type PaymentSecurityMonitor struct {
    alerting AlertService
}

func (m *PaymentSecurityMonitor) DetectAnomalies(payment *Payment) {
    // Velocity checks
    if m.isHighVelocity(payment.CustomerID) {
        m.alerting.SendAlert(Alert{
            Type:     "high_velocity_payments",
            Severity: "medium",
            Details:  "Multiple payments from same customer",
        })
    }
    
    // Amount threshold checks
    if payment.Amount > LARGE_PAYMENT_THRESHOLD {
        m.alerting.SendAlert(Alert{
            Type:     "large_payment",
            Severity: "info",
            Details:  fmt.Sprintf("Payment over threshold: %d", payment.Amount),
        })
    }
    
    // Geographic anomalies
    if m.isGeographicAnomaly(payment) {
        m.alerting.SendAlert(Alert{
            Type:     "geographic_anomaly",
            Severity: "high",
            Details:  "Payment from unusual location",
        })
    }
}
```

✅ payment-sentinel completed secure payment integration

PaymentSentinel: Key security measures implemented:

1. **PCI Compliance**:
   - No card data storage - only Stripe tokens
   - TLS 1.2+ enforcement
   - Encrypted audit logs

2. **Webhook Security**:
   - Signature verification on all webhooks
   - Idempotency to prevent replay attacks
   - Asynchronous processing

3. **Monitoring**:
   - Velocity checking
   - Anomaly detection
   - Comprehensive audit trail

4. **Error Handling**:
   - Never expose internal errors
   - Rate limiting on all endpoints
   - Graceful degradation

Remember: Security is not a feature, it's a requirement!
EOF

echo ""
echo "📊 Payment Integration Summary:"
echo "- Implemented PCI-compliant payment flow"
echo "- Secured webhook handling with signature verification"
echo "- Added fraud detection and monitoring"
echo "- Created comprehensive audit logging"
echo ""
echo "🎯 PaymentSentinel's security expertise includes:"
echo "- PCI DSS compliance requirements"
echo "- Secure payment tokenization"
echo "- Webhook security best practices"
echo "- Fraud prevention strategies"