package ui

import "time"

// RingBuffer is a fixed-size circular buffer for chart data.
type RingBuffer struct {
	data     []ChartDataPoint
	head     int
	size     int
	capacity int
}

// NewRingBuffer creates a new ring buffer with the given capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		data:     make([]ChartDataPoint, capacity),
		capacity: capacity,
	}
}

// Push adds a new data point to the buffer.
func (r *RingBuffer) Push(timestamp time.Time, value float64) {
	r.data[r.head] = ChartDataPoint{
		Timestamp:  timestamp,
		Throughput: value,
	}
	r.head = (r.head + 1) % r.capacity
	if r.size < r.capacity {
		r.size++
	}
}

// Values returns all values in chronological order.
func (r *RingBuffer) Values() []float64 {
	if r.size == 0 {
		return nil
	}

	result := make([]float64, r.size)
	start := 0
	if r.size == r.capacity {
		start = r.head
	}

	for i := 0; i < r.size; i++ {
		idx := (start + i) % r.capacity
		result[i] = r.data[idx].Throughput
	}
	return result
}

// DataPoints returns all data points in chronological order.
func (r *RingBuffer) DataPoints() []ChartDataPoint {
	if r.size == 0 {
		return nil
	}

	result := make([]ChartDataPoint, r.size)
	start := 0
	if r.size == r.capacity {
		start = r.head
	}

	for i := 0; i < r.size; i++ {
		idx := (start + i) % r.capacity
		result[i] = r.data[idx]
	}
	return result
}

// Last returns the most recent value or 0 if empty.
func (r *RingBuffer) Last() float64 {
	if r.size == 0 {
		return 0
	}
	idx := r.head - 1
	if idx < 0 {
		idx = r.capacity - 1
	}
	return r.data[idx].Throughput
}

// Len returns the number of elements in the buffer.
func (r *RingBuffer) Len() int {
	return r.size
}

// Clear resets the buffer.
func (r *RingBuffer) Clear() {
	r.head = 0
	r.size = 0
}

// Average returns the average of all values.
func (r *RingBuffer) Average() float64 {
	if r.size == 0 {
		return 0
	}
	var sum float64
	for _, v := range r.Values() {
		sum += v
	}
	return sum / float64(r.size)
}

// Max returns the maximum value.
func (r *RingBuffer) Max() float64 {
	if r.size == 0 {
		return 0
	}
	max := r.data[0].Throughput
	for _, v := range r.Values() {
		if v > max {
			max = v
		}
	}
	return max
}
