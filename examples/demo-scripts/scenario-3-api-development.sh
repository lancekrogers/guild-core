#!/bin/bash
# Scenario 3: API Development with ServiceArchitect
# Duration: 4-5 minutes
# Purpose: Deep dive into API development with specialized agent

echo "🏰 Guild Framework Demo - Scenario 3: API Development"
echo "===================================================="
echo ""
echo "🏗️ Working with ServiceArchitect on product catalog API..."
echo ""

# Show chat interaction
echo "$ guild chat --campaign e-commerce"
echo ""
echo "You: @service-architect Design a complete REST API for the product catalog with search, filtering, and pagination"
echo ""
sleep 1

cat << 'EOF'
🤔 service-architect is thinking...
   Analyzing requirements for product catalog API...

ServiceArchitect: I'll design a comprehensive REST API for the product catalog with advanced search, filtering, and pagination capabilities. Let me create a production-ready solution.

⚙️ service-architect is working...
   Designing API structure...
   [████░░░░░░░░░░░░] 25%

## Product Catalog API Design

### Base URL
```
https://api.ecommerce.com/v1
```

### Endpoints Overview

⚙️ service-architect is working...
   Creating endpoint specifications...
   [████████░░░░░░░░] 50%

#### 1. List Products with Search and Filters
```
GET /products
```

**Query Parameters:**
- `q` (string): Full-text search query
- `category` (string): Filter by category ID
- `brand` (string[]): Filter by brand names
- `price_min` (number): Minimum price
- `price_max` (number): Maximum price
- `in_stock` (boolean): Only show in-stock items
- `sort` (string): Sort field (price_asc, price_desc, name, rating, created_at)
- `page` (number): Page number (default: 1)
- `limit` (number): Items per page (default: 20, max: 100)

**Response:**
```json
{
  "data": [
    {
      "id": "prod_123",
      "name": "Premium Wireless Headphones",
      "slug": "premium-wireless-headphones",
      "description": "High-quality wireless headphones...",
      "price": 299.99,
      "currency": "USD",
      "brand": "AudioTech",
      "category": {
        "id": "cat_456",
        "name": "Electronics",
        "slug": "electronics"
      },
      "images": [
        {
          "url": "https://cdn.example.com/products/123/main.jpg",
          "alt": "Product main image",
          "is_primary": true
        }
      ],
      "stock": {
        "quantity": 50,
        "status": "in_stock"
      },
      "rating": {
        "average": 4.5,
        "count": 234
      },
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-20T15:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 156,
    "pages": 8
  },
  "filters": {
    "categories": [
      {"id": "cat_456", "name": "Electronics", "count": 45}
    ],
    "brands": [
      {"name": "AudioTech", "count": 12}
    ],
    "price_range": {
      "min": 19.99,
      "max": 999.99
    }
  }
}
```

⚙️ service-architect is working...
   Implementing search logic...
   [████████████░░░░] 75%

#### 2. Get Product Details
```
GET /products/{id}
```

#### 3. Create Product (Admin)
```
POST /products
Authorization: Bearer {admin_token}
```

**Request Body:**
```json
{
  "name": "New Product",
  "description": "Product description",
  "price": 99.99,
  "category_id": "cat_456",
  "brand": "BrandName",
  "sku": "SKU-12345",
  "stock_quantity": 100,
  "images": [
    {
      "url": "https://example.com/image.jpg",
      "is_primary": true
    }
  ],
  "specifications": {
    "weight": "500g",
    "dimensions": "10x10x5cm"
  }
}
```

### Implementation Code

```go
// catalog-service/api/handlers/product.go
package handlers

import (
    "net/http"
    "strconv"
    
    "github.com/gin-gonic/gin"
    "github.com/ecommerce/catalog-service/models"
    "github.com/ecommerce/catalog-service/services"
)

type ProductHandler struct {
    productService *services.ProductService
    searchService  *services.SearchService
}

func (h *ProductHandler) ListProducts(c *gin.Context) {
    // Parse query parameters
    params := parseProductParams(c)
    
    // Execute search with filters
    result, err := h.searchService.SearchProducts(c.Request.Context(), params)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Search failed",
        })
        return
    }
    
    // Get aggregations for filters
    filters, err := h.searchService.GetFilterAggregations(c.Request.Context(), params)
    if err != nil {
        // Log error but don't fail the request
        filters = &models.FilterAggregations{}
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data": result.Products,
        "pagination": gin.H{
            "page": params.Page,
            "limit": params.Limit,
            "total": result.Total,
            "pages": (result.Total + params.Limit - 1) / params.Limit,
        },
        "filters": filters,
    })
}

func parseProductParams(c *gin.Context) *models.SearchParams {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    
    if limit > 100 {
        limit = 100
    }
    
    return &models.SearchParams{
        Query:     c.Query("q"),
        Category:  c.Query("category"),
        Brands:    c.QueryArray("brand"),
        PriceMin:  parseFloat(c.Query("price_min")),
        PriceMax:  parseFloat(c.Query("price_max")),
        InStock:   c.Query("in_stock") == "true",
        Sort:      c.DefaultQuery("sort", "created_at"),
        Page:      page,
        Limit:     limit,
    }
}
```

### PostgreSQL Search Implementation

```sql
-- Full-text search with PostgreSQL
CREATE INDEX idx_products_search ON products 
USING GIN(to_tsvector('english', name || ' ' || description || ' ' || COALESCE(brand, '')));

-- Search query with filters
WITH filtered_products AS (
    SELECT p.*, 
           ts_rank(to_tsvector('english', p.name || ' ' || p.description), 
                   plainto_tsquery('english', $1)) as rank
    FROM products p
    WHERE ($1 = '' OR to_tsvector('english', p.name || ' ' || p.description) @@ plainto_tsquery('english', $1))
      AND ($2::uuid IS NULL OR p.category_id = $2)
      AND ($3::text[] IS NULL OR p.brand = ANY($3))
      AND ($4::numeric IS NULL OR p.price >= $4)
      AND ($5::numeric IS NULL OR p.price <= $5)
      AND ($6::boolean IS FALSE OR p.stock_quantity > 0)
)
SELECT * FROM filtered_products
ORDER BY 
    CASE WHEN $7 = 'price_asc' THEN price END ASC,
    CASE WHEN $7 = 'price_desc' THEN price END DESC,
    CASE WHEN $7 = 'name' THEN name END ASC,
    CASE WHEN $7 = 'rating' THEN rating END DESC,
    CASE WHEN $1 != '' THEN rank END DESC,
    created_at DESC
LIMIT $8 OFFSET $9;
```

✅ service-architect completed API design

ServiceArchitect: I've also prepared the OpenAPI specification:

```yaml
# catalog-service/api/openapi.yaml
openapi: 3.0.0
info:
  title: Product Catalog API
  version: 1.0.0
  description: E-commerce product catalog with advanced search

paths:
  /products:
    get:
      summary: List products with search and filters
      parameters:
        - name: q
          in: query
          schema:
            type: string
          description: Search query
        # ... other parameters
      responses:
        200:
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ProductListResponse'
```

EOF

echo ""
echo "📊 API Development Summary:"
echo "- Designed RESTful endpoints with advanced features"
echo "- Implemented full-text search with PostgreSQL"
echo "- Created pagination and filtering system"
echo "- Provided OpenAPI documentation"
echo "- Included performance optimizations"
echo ""
echo "🎯 This demonstrates ServiceArchitect's expertise in:"
echo "- API design best practices"
echo "- Database query optimization"
echo "- Search implementation strategies"
echo "- Production-ready code generation"