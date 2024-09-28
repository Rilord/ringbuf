package bench

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/BurntSushi/toml"
	ringbuf "github.com/Rilord/ringbuf"
)

const (
	defaultBenchConfigPath = `values_bench.toml`
	BufferSizeEnv          = `BUFFER_SIZE`
	MaxThreadsEnv          = `MAX_THREADS`
	ProducersCountEnv      = `PRODUCERS_COUNT`
)

type BenchConfig struct {
	BufferSize     int
	MaxThreads     int
	ProducersCount int
}

type SmallStruct struct {
	x, y, z float32
}

func toInt(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse %s, check ENV variables", s))
	}
	return v
}

// Get Bench
func setup() BenchConfig {
	_, err := os.Stat(defaultBenchConfigPath)

	if os.IsNotExist(err) {
		panic("couldn't find config file")
	} else if err != nil {
		panic(err)
	}

	var cfg BenchConfig
	_, err = toml.DecodeFile(defaultBenchConfigPath, &cfg)
	if err != nil {
		panic("couldn't decode toml")
	}

	if val := os.Getenv(BufferSizeEnv); val != "" {
		cfg.BufferSize = toInt(val)
	}
	if val := os.Getenv(MaxThreadsEnv); val != "" {
		cfg.MaxThreads = toInt(val)
	}
	if val := os.Getenv(ProducersCountEnv); val != "" {
		cfg.ProducersCount = toInt(val)
	}
	return cfg
}

func spawnProducers(producerCh chan<- bool, numThreads int, numProducers int) {
	runtime.GOMAXPROCS(numThreads)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range numThreads {
			if numProducers > 0 {
				producerCh <- true
				numProducers--
			} else {
				producerCh <- false

			}
		}
	}()
	wg.Wait()
}

func generateBenchData(size int) []SmallStruct {
	data := make([]SmallStruct, size)
	for i := range size {
		data[i].x = float32(i)
		data[i].y = float32(i * i)
		data[i].z = float32(i * i * i)
	}

	return data
}

func BenchmarkRingBuffer(b *testing.B) {

	handoversCount := int32(0)

	conf := setup()
	producerCh := make(chan bool, conf.MaxThreads)
	spawnProducers(producerCh, conf.MaxThreads, conf.ProducersCount)
	data := generateBenchData(10000)
	buf := make([]SmallStruct, 10000)
	rb := ringbuf.NewRingBuffer[SmallStruct](int(conf.BufferSize))

	b.RunParallel(func(p *testing.PB) {
		isProducer := <-producerCh
		for i := 1; p.Next(); i++ {
			if isProducer {
				rb.Write(data[(i % (len(data) - 1))])
			} else {
				read := rb.ReadVec(buf)
				atomic.AddInt32(&handoversCount, int32(read))
			}

		}
	})

	b.StopTimer()
	b.ReportMetric(float64(handoversCount), "handovers")
}

func BenchmarkRingBufferWithoutAlignment(b *testing.B) {

	handoversCount := int32(0)

	conf := setup()
	producerCh := make(chan bool, conf.MaxThreads)
	spawnProducers(producerCh, conf.MaxThreads, conf.ProducersCount)
	data := generateBenchData(10000)
	buf := make([]SmallStruct, 10000)
	rb := ringbuf.NewRingBufferWithoutAlign[SmallStruct](int(conf.BufferSize))

	b.RunParallel(func(p *testing.PB) {
		isProducer := <-producerCh
		for i := 1; p.Next(); i++ {
			if isProducer {
				rb.Write(data[(i % (len(data) - 1))])
			} else {
				read := rb.ReadVec(buf)
				atomic.AddInt32(&handoversCount, int32(read))
			}

		}
	})

	b.StopTimer()
	b.ReportMetric(float64(handoversCount), "handovers")
}
