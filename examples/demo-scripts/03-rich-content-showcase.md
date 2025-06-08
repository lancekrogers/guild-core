# Rich Content Showcase Demo (3 minutes)

**Goal**: Demonstrate Guild's superior visual rendering capabilities and rich content presentation compared to plain-text competitors

## Pre-Demo Setup

```bash
cd /path/to/demo-workspace
guild init
cp ../../.guild/commissions/e-commerce-platform.md .guild/commissions/
cp ../../.guild/guild.yaml .guild/
```

## Demo Script

### Opening: The Visual Difference (0-30 seconds) 🎨

**Narrator**: "While other AI coding tools show plain text, Guild renders rich, professional content that makes development more intuitive and impressive."

**Commands**:

```bash
# Start with the commission to show rich markdown
guild chat --campaign e-commerce
```

**In Chat**:

```
Show me our e-commerce commission requirements with technical specifications
```

**Expected Display**: Rich markdown with headers, tables, emphasis, and structured content

### Segment 1: Code Syntax Highlighting (30-90 seconds) 🌈

**Chat Commands**:

```
@backend-specialist Show me a complete Go microservice implementation with error handling, middleware, and database integration
```

**Expected Rich Response**:

```go
// E-commerce Product Service Implementation
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/jmoiron/sqlx"
    "github.com/lib/pq"
    _ "github.com/lib/pq"
)

// Product represents a product in our e-commerce system
type Product struct {
    ID          string    `json:"id" db:"id"`
    Name        string    `json:"name" db:"name" validate:"required,min=1,max=255"`
    Description string    `json:"description" db:"description"`
    Price       float64   `json:"price" db:"price" validate:"required,gt=0"`
    Inventory   int       `json:"inventory" db:"inventory_quantity" validate:"gte=0"`
    CategoryID  string    `json:"category_id" db:"category_id" validate:"required,uuid"`
    Status      string    `json:"status" db:"status"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ProductService handles all product-related operations
type ProductService struct {
    db     *sqlx.DB
    logger *log.Logger
}

// NewProductService creates a new product service
func NewProductService(db *sqlx.DB, logger *log.Logger) *ProductService {
    return &ProductService{
        db:     db,
        logger: logger,
    }
}

// GetProducts retrieves products with pagination and filtering
func (s *ProductService) GetProducts(ctx context.Context, filters ProductFilters) (*ProductResponse, error) {
    query := `
        SELECT p.id, p.name, p.description, p.price, p.inventory_quantity,
               p.category_id, p.status, p.created_at, p.updated_at,
               c.name as category_name
        FROM products p
        LEFT JOIN categories c ON p.category_id = c.id
        WHERE p.status = 'active'
    `

    args := []interface{}{}
    argIndex := 1

    // Add search filter
    if filters.Search != "" {
        query += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.description ILIKE $%d)", argIndex, argIndex+1)
        searchTerm := "%" + filters.Search + "%"
        args = append(args, searchTerm, searchTerm)
        argIndex += 2
    }

    // Add category filter
    if filters.CategoryID != "" {
        query += fmt.Sprintf(" AND p.category_id = $%d", argIndex)
        args = append(args, filters.CategoryID)
        argIndex++
    }

    // Add price range filter
    if filters.MinPrice > 0 {
        query += fmt.Sprintf(" AND p.price >= $%d", argIndex)
        args = append(args, filters.MinPrice)
        argIndex++
    }

    if filters.MaxPrice > 0 {
        query += fmt.Sprintf(" AND p.price <= $%d", argIndex)
        args = append(args, filters.MaxPrice)
        argIndex++
    }

    // Add sorting
    if filters.SortBy != "" {
        query += " ORDER BY " + filters.SortBy
        if filters.SortOrder == "desc" {
            query += " DESC"
        } else {
            query += " ASC"
        }
    } else {
        query += " ORDER BY p.created_at DESC"
    }

    // Add pagination
    query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
    args = append(args, filters.Limit, filters.Offset)

    var products []Product
    if err := s.db.SelectContext(ctx, &products, query, args...); err != nil {
        s.logger.Printf("Error fetching products: %v", err)
        return nil, fmt.Errorf("failed to fetch products: %w", err)
    }

    // Get total count for pagination
    countQuery := `SELECT COUNT(*) FROM products WHERE status = 'active'`
    var totalCount int
    if err := s.db.GetContext(ctx, &totalCount, countQuery); err != nil {
        s.logger.Printf("Error counting products: %v", err)
        return nil, fmt.Errorf("failed to count products: %w", err)
    }

    return &ProductResponse{
        Products: products,
        Pagination: PaginationInfo{
            Page:       filters.Page,
            Limit:      filters.Limit,
            Total:      totalCount,
            TotalPages: (totalCount + filters.Limit - 1) / filters.Limit,
            HasNext:    filters.Offset+filters.Limit < totalCount,
            HasPrev:    filters.Page > 1,
        },
    }, nil
}

// CreateProduct creates a new product
func (s *ProductService) CreateProduct(ctx context.Context, product *Product) error {
    // Generate UUID for new product
    product.ID = generateUUID()
    product.CreatedAt = time.Now()
    product.UpdatedAt = time.Now()
    product.Status = "active"

    query := `
        INSERT INTO products (id, name, description, price, inventory_quantity, category_id, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `

    _, err := s.db.ExecContext(ctx, query,
        product.ID, product.Name, product.Description, product.Price,
        product.Inventory, product.CategoryID, product.Status,
        product.CreatedAt, product.UpdatedAt,
    )

    if err != nil {
        if pqErr, ok := err.(*pq.Error); ok {
            switch pqErr.Code {
            case "23505": // unique_violation
                return fmt.Errorf("product with this name already exists: %w", err)
            case "23503": // foreign_key_violation
                return fmt.Errorf("invalid category ID: %w", err)
            }
        }
        s.logger.Printf("Error creating product: %v", err)
        return fmt.Errorf("failed to create product: %w", err)
    }

    s.logger.Printf("Product created successfully: %s", product.ID)
    return nil
}

// Server represents our HTTP server
type Server struct {
    productService *ProductService
    router         *gin.Engine
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
    // Middleware
    s.router.Use(gin.Logger())
    s.router.Use(gin.Recovery())
    s.router.Use(corsMiddleware())
    s.router.Use(rateLimitMiddleware())

    // Health check
    s.router.GET("/health", s.healthCheck)

    // API routes
    v1 := s.router.Group("/api/v1")
    {
        // Product routes
        products := v1.Group("/products")
        {
            products.GET("", s.getProducts)
            products.GET("/:id", s.getProduct)
            products.POST("", authMiddleware(), adminMiddleware(), s.createProduct)
            products.PUT("/:id", authMiddleware(), adminMiddleware(), s.updateProduct)
            products.DELETE("/:id", authMiddleware(), adminMiddleware(), s.deleteProduct)
        }

        // Search routes
        v1.GET("/search", s.searchProducts)
    }
}

// HTTP Handlers
func (s *Server) getProducts(c *gin.Context) {
    filters := ProductFilters{
        Search:     c.Query("search"),
        CategoryID: c.Query("category"),
        SortBy:     c.DefaultQuery("sort_by", "created_at"),
        SortOrder:  c.DefaultQuery("sort_order", "desc"),
        Page:       1,
        Limit:      20,
    }

    if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && page > 0 {
        filters.Page = page
    }

    if limit, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && limit > 0 && limit <= 100 {
        filters.Limit = limit
    }

    filters.Offset = (filters.Page - 1) * filters.Limit

    if minPrice, err := strconv.ParseFloat(c.Query("min_price"), 64); err == nil {
        filters.MinPrice = minPrice
    }

    if maxPrice, err := strconv.ParseFloat(c.Query("max_price"), 64); err == nil {
        filters.MaxPrice = maxPrice
    }

    response, err := s.productService.GetProducts(c.Request.Context(), filters)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": map[string]interface{}{
                "code":    "INTERNAL_ERROR",
                "message": "Failed to fetch products",
                "details": err.Error(),
            },
        })
        return
    }

    c.JSON(http.StatusOK, response)
}
```

**Visual Focus**: Highlight the beautiful syntax highlighting, proper indentation, and professional code structure

### Segment 2: Multi-Language Support (90-150 seconds) 💻

**Chat Commands**:

```
@frontend-specialist Show me React components with TypeScript, @devops-specialist show Docker configurations, @qa-specialist show testing code
```

**Expected Parallel Rich Responses**:

**TypeScript/React**:

```typescript
// Advanced React Component with Hooks and TypeScript
import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from 'react-query';
import { debounce } from 'lodash';

interface Product {
  id: string;
  name: string;
  description: string;
  price: number;
  inventory: number;
  category: Category;
  images: string[];
  ratings: {
    average: number;
    count: number;
  };
}

interface ProductCatalogProps {
  categoryId?: string;
  searchQuery?: string;
  onProductSelect: (product: Product) => void;
}

export const ProductCatalog: React.FC<ProductCatalogProps> = ({
  categoryId,
  searchQuery: initialSearchQuery = '',
  onProductSelect,
}) => {
  const [searchQuery, setSearchQuery] = useState(initialSearchQuery);
  const [filters, setFilters] = useState<ProductFilters>({
    category: categoryId,
    priceRange: { min: 0, max: 1000 },
    sortBy: 'popularity',
    sortOrder: 'desc',
  });
  const [page, setPage] = useState(1);

  const queryClient = useQueryClient();

  // Debounced search to avoid excessive API calls
  const debouncedSearch = useCallback(
    debounce((query: string) => {
      setSearchQuery(query);
      setPage(1); // Reset to first page on new search
    }, 300),
    []
  );

  // Fetch products with React Query
  const {
    data: productsResponse,
    isLoading,
    isError,
    error,
  } = useQuery(
    ['products', { searchQuery, filters, page }],
    () => productApi.getProducts({
      search: searchQuery,
      ...filters,
      page,
      limit: 20
    }),
    {
      keepPreviousData: true,
      staleTime: 5 * 60 * 1000, // 5 minutes
      cacheTime: 10 * 60 * 1000, // 10 minutes
    }
  );

  // Add to cart mutation
  const addToCartMutation = useMutation(
    (data: { productId: string; quantity: number }) =>
      cartApi.addItem(data.productId, data.quantity),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(['cart']);
        toast.success('Product added to cart!');
      },
      onError: (error: any) => {
        toast.error(error.message || 'Failed to add product to cart');
      },
    }
  );

  const handleAddToCart = useCallback((product: Product, quantity: number = 1) => {
    addToCartMutation.mutate({ productId: product.id, quantity });
  }, [addToCartMutation]);

  // Memoized product grid to optimize rendering
  const productGrid = useMemo(() => {
    if (!productsResponse?.products) return null;

    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        {productsResponse.products.map((product) => (
          <ProductCard
            key={product.id}
            product={product}
            onSelect={() => onProductSelect(product)}
            onAddToCart={(quantity) => handleAddToCart(product, quantity)}
            isAddingToCart={addToCartMutation.isLoading}
          />
        ))}
      </div>
    );
  }, [productsResponse?.products, onProductSelect, handleAddToCart, addToCartMutation.isLoading]);

  if (isError) {
    return (
      <ErrorBoundary>
        <div className="text-center py-12">
          <AlertTriangle className="mx-auto h-12 w-12 text-red-500" />
          <h3 className="mt-2 text-lg font-medium text-gray-900">
            Unable to load products
          </h3>
          <p className="mt-1 text-gray-500">
            {error instanceof Error ? error.message : 'Something went wrong'}
          </p>
          <button
            onClick={() => queryClient.invalidateQueries(['products'])}
            className="mt-4 bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
          >
            Try Again
          </button>
        </div>
      </ErrorBoundary>
    );
  }

  return (
    <div className="product-catalog">
      {/* Search and Filters */}
      <div className="mb-8 space-y-4">
        <SearchInput
          value={searchQuery}
          onChange={debouncedSearch}
          placeholder="Search products..."
          className="w-full"
        />

        <ProductFilters
          filters={filters}
          onFiltersChange={setFilters}
          categories={categories}
        />
      </div>

      {/* Loading State */}
      {isLoading && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {Array.from({ length: 8 }).map((_, i) => (
            <ProductCardSkeleton key={i} />
          ))}
        </div>
      )}

      {/* Products Grid */}
      {!isLoading && productGrid}

      {/* Pagination */}
      {productsResponse?.pagination && (
        <Pagination
          currentPage={page}
          totalPages={productsResponse.pagination.totalPages}
          onPageChange={setPage}
          className="mt-8"
        />
      )}
    </div>
  );
};
```

**Docker**:

```dockerfile
# Multi-stage Docker build for React application
FROM node:18-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./
COPY yarn.lock ./

# Install dependencies
RUN yarn install --frozen-lockfile

# Copy source code
COPY . .

# Build the application
RUN yarn build

# Production stage
FROM nginx:alpine

# Copy built assets from builder stage
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy custom nginx configuration
COPY nginx.conf /etc/nginx/nginx.conf

# Expose port 80
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
```

**Testing Code**:

```go
// Comprehensive testing suite
func TestProductService_GetProducts(t *testing.T) {
    tests := []struct {
        name           string
        filters        ProductFilters
        setupMock      func(*sql.MockDB)
        expectedResult *ProductResponse
        expectedError  string
    }{
        {
            name: "successful retrieval with pagination",
            filters: ProductFilters{
                Search:    "laptop",
                CategoryID: "electronics",
                Page:      1,
                Limit:     10,
            },
            setupMock: func(mock *sql.MockDB) {
                mock.ExpectQuery("SELECT (.+) FROM products").
                    WithArgs("laptop", "laptop", "electronics", 10, 0).
                    WillReturnRows(sqlmock.NewRows([]string{
                        "id", "name", "price", "inventory_quantity",
                    }).AddRow(
                        "123", "MacBook Pro", 2499.99, 5,
                    ))

                mock.ExpectQuery("SELECT COUNT").
                    WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))
            },
            expectedResult: &ProductResponse{
                Products: []Product{{
                    ID: "123", Name: "MacBook Pro", Price: 2499.99, Inventory: 5,
                }},
                Pagination: PaginationInfo{Page: 1, Limit: 10, Total: 25},
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            db, mock, err := sqlmock.New()
            require.NoError(t, err)
            defer db.Close()

            tt.setupMock(mock)

            service := NewProductService(sqlx.NewDb(db, "postgres"), log.New(os.Stdout, "", 0))
            result, err := service.GetProducts(context.Background(), tt.filters)

            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expectedResult, result)
            }

            assert.NoError(t, mock.ExpectationsWereMet())
        })
    }
}
```

### Segment 3: Documentation and Diagrams (150-180 seconds) 📊

**Chat Commands**:

```
@documentation-specialist Create comprehensive API documentation with examples, and @service-architect show me system architecture diagrams
```

**Expected Response**: Rich API documentation with properly formatted tables, code examples, and potentially rendered Mermaid diagrams

## Closing: The Competitive Advantage (150-180 seconds) 🚀

**Narrator**: "While competitors show plain text responses, Guild delivers a rich, visual development experience that makes complex projects manageable and impressive. This is the future of AI-assisted development."

**Visual Summary**:

- Side-by-side comparison concepts (implied)
- Professional syntax highlighting throughout
- Rich markdown rendering
- Multi-language support demonstration
- Professional development tool appearance

## Recording Notes

### Technical Setup

- **Focus**: Maximize syntax highlighting visibility
- **Languages Showcased**: Go, TypeScript, SQL, Docker, YAML, Bash
- **Visual Elements**: Headers, tables, code blocks, emphasis
- **Theme**: High contrast for clear syntax highlighting

### Key Visual Moments

1. **Rich Markdown**: Professional document formatting
2. **Syntax Highlighting**: Multiple programming languages
3. **Code Structure**: Proper indentation and organization
4. **Multi-Language**: Parallel responses in different languages
5. **Documentation**: API docs with examples and tables

### Success Criteria

- ✅ Clear visual superiority over plain-text tools
- ✅ Professional development environment appearance
- ✅ Rich content rendering without glitches
- ✅ Multiple programming languages highlighted
- ✅ Documentation and diagrams properly formatted

This demo specifically targets the visual competitive advantage that Guild provides in the AI coding assistant space.
