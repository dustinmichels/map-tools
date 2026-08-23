package strava

// Activity is a row from activities.csv.
type Activity struct {
	ActivityID          int64
	ActivityDate        string
	ActivityName        string
	ActivityType        string
	ActivityDescription string
	ElapsedTime         *int64
	Distance            *float64
	Filename            *string
	MovingTime          *int64
	MaxSpeed            *float64
	AverageSpeed        *float64
	ElevationGain       *float64
	ElevationLoss       *float64
	AverageHeartRate    *string
	ActivityClass       *string // "Type" column
	StartTime           *string
	CarbonSaved         *int64
	Media               *string
}

// ActivityRow is one row in the output geoparquet file.
// Geometry is WKB-encoded (little-endian) LineString: lon/lat pairs.
type ActivityRow struct {
	ActivityID       int64   `parquet:"activity_id"`
	ActivityDate     string  `parquet:"activity_date"`
	ActivityName     string  `parquet:"activity_name"`
	ActivityType     string  `parquet:"activity_type"`
	ElapsedTime      int64   `parquet:"elapsed_time"`
	DistanceKm       float64 `parquet:"distance_km"`
	MovingTime       int64   `parquet:"moving_time"`
	MaxSpeed         float64 `parquet:"max_speed"`
	AverageSpeed     float64 `parquet:"average_speed"`
	ElevationGain    float64 `parquet:"elevation_gain"`
	ElevationLoss    float64 `parquet:"elevation_loss"`
	AverageHeartRate string  `parquet:"average_heart_rate"`
	Filename         string  `parquet:"filename"`
	Geometry         []byte  `parquet:"geometry"` // WKB LineString
}
