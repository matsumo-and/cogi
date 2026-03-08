package vector

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

// Point represents a vector point in Qdrant
type Point struct {
	ID        string
	Vector    []float32
	Payload   map[string]interface{}
	SymbolID  int64
	Granularity string
}

// SearchResult represents a search result from Qdrant
type SearchResult struct {
	ID       string
	Score    float32
	Payload  map[string]interface{}
	SymbolID int64
}

// Client wraps the Qdrant client
type Client struct {
	client         *qdrant.Client
	collectionName string
	dimension      uint64
}

// NewClient creates a new Qdrant vector client
func NewClient(endpoint, collectionName string, dimension int) (*Client, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: endpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	return &Client{
		client:         client,
		collectionName: collectionName,
		dimension:      uint64(dimension),
	}, nil
}

// EnsureCollection creates the collection if it doesn't exist
func (c *Client) EnsureCollection(ctx context.Context) error {
	// Check if collection exists
	exists, err := c.client.CollectionExists(ctx, c.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		return nil
	}

	// Create collection
	err = c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: c.collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     c.dimension,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// Create payload indexes for efficient filtering
	payloadIndexes := []string{"symbol_id", "granularity", "repository_id", "language", "symbol_kind"}
	for _, field := range payloadIndexes {
		_, err := c.client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: c.collectionName,
			FieldName:      field,
			FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
		})
		if err != nil {
			// Ignore errors if index already exists
			continue
		}
	}

	return nil
}

// UpsertPoint adds or updates a single point
func (c *Client) UpsertPoint(ctx context.Context, point Point) error {
	return c.UpsertPoints(ctx, []Point{point})
}

// UpsertPoints adds or updates multiple points in batch
func (c *Client) UpsertPoints(ctx context.Context, points []Point) error {
	if len(points) == 0 {
		return nil
	}

	qdrantPoints := make([]*qdrant.PointStruct, 0, len(points))
	for _, p := range points {
		pointID := p.ID
		if pointID == "" {
			pointID = uuid.New().String()
		}

		qdrantPoints = append(qdrantPoints, &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(pointID),
			Vectors: qdrant.NewVectors(p.Vector...),
			Payload: qdrant.NewValueMap(p.Payload),
		})
	}

	_, err := c.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: c.collectionName,
		Points:         qdrantPoints,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

// DeletePoint deletes a point by ID
func (c *Client) DeletePoint(ctx context.Context, pointID string) error {
	return c.DeletePoints(ctx, []string{pointID})
}

// DeletePoints deletes multiple points by IDs
func (c *Client) DeletePoints(ctx context.Context, pointIDs []string) error {
	if len(pointIDs) == 0 {
		return nil
	}

	ids := make([]*qdrant.PointId, 0, len(pointIDs))
	for _, id := range pointIDs {
		ids = append(ids, qdrant.NewIDUUID(id))
	}

	_, err := c.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: c.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: ids,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete points: %w", err)
	}

	return nil
}

// Search performs vector similarity search
func (c *Client) Search(ctx context.Context, vector []float32, limit int, filter map[string]interface{}) ([]SearchResult, error) {
	// Build filter if provided
	var qdrantFilter *qdrant.Filter
	if len(filter) > 0 {
		conditions := make([]*qdrant.Condition, 0, len(filter))
		for key, value := range filter {
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: key,
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
							Keyword: fmt.Sprintf("%v", value),
							},
						},
					},
				},
			})
		}
		qdrantFilter = &qdrant.Filter{
			Must: conditions,
		}
	}

	searchResult, err := c.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: c.collectionName,
		Query:          qdrant.NewQuery(vector...),
		Limit:          qdrant.PtrOf(uint64(limit)),
		Filter:         qdrantFilter,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	results := make([]SearchResult, 0, len(searchResult))
	for _, hit := range searchResult {
		payload := make(map[string]interface{})
		if hit.Payload != nil {
			for k, v := range hit.Payload {
				payload[k] = extractValue(v)
			}
		}

		symbolID := int64(0)
		if sid, ok := payload["symbol_id"]; ok {
			if sidInt, ok := sid.(int64); ok {
				symbolID = sidInt
			}
		}

		results = append(results, SearchResult{
			ID:       hit.Id.GetUuid(),
			Score:    hit.Score,
			Payload:  payload,
			SymbolID: symbolID,
		})
	}

	return results, nil
}

// extractValue extracts the actual value from Qdrant Value type
func extractValue(v *qdrant.Value) interface{} {
	if v == nil {
		return nil
	}

	switch kind := v.Kind.(type) {
	case *qdrant.Value_IntegerValue:
		return kind.IntegerValue
	case *qdrant.Value_DoubleValue:
		return kind.DoubleValue
	case *qdrant.Value_StringValue:
		return kind.StringValue
	case *qdrant.Value_BoolValue:
		return kind.BoolValue
	case *qdrant.Value_ListValue:
		list := make([]interface{}, len(kind.ListValue.Values))
		for i, item := range kind.ListValue.Values {
			list[i] = extractValue(item)
		}
		return list
	case *qdrant.Value_StructValue:
		m := make(map[string]interface{})
		for k, val := range kind.StructValue.Fields {
			m[k] = extractValue(val)
		}
		return m
	default:
		return nil
	}
}

// DeleteBySymbolID deletes all points associated with a symbol ID
func (c *Client) DeleteBySymbolID(ctx context.Context, symbolID int64) error {
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "symbol_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Integer{
								Integer: symbolID,
							},
						},
					},
				},
			},
		},
	}

	_, err := c.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: c.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: filter,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete points by symbol ID: %w", err)
	}

	return nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
