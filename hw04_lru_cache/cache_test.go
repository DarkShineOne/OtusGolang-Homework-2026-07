package hw04lrucache

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	t.Run("empty cache", func(t *testing.T) {
		c := NewCache(10)

		_, ok := c.Get("aaa")
		require.False(t, ok)

		_, ok = c.Get("bbb")
		require.False(t, ok)
	})

	t.Run("simple", func(t *testing.T) {
		c := NewCache(5)

		wasInCache := c.Set("aaa", 100)
		require.False(t, wasInCache)

		wasInCache = c.Set("bbb", 200)
		require.False(t, wasInCache)

		val, ok := c.Get("aaa")
		require.True(t, ok)
		require.Equal(t, 100, val)

		val, ok = c.Get("bbb")
		require.True(t, ok)
		require.Equal(t, 200, val)

		wasInCache = c.Set("aaa", 300)
		require.True(t, wasInCache)

		val, ok = c.Get("aaa")
		require.True(t, ok)
		require.Equal(t, 300, val)

		val, ok = c.Get("ccc")
		require.False(t, ok)
		require.Nil(t, val)
	})

	// выталкивание из-за емкости
	t.Run("capacity overflow", func(t *testing.T) {
		c := NewCache(3)
		c.Set("A", 1)
		c.Set("B", 2)
		c.Set("C", 3)
		c.Set("D", 4)

		val, ok := c.Get("A")
		require.False(t, ok)
		require.Nil(t, val)

		val, ok = c.Get("D")
		require.True(t, ok)
		require.Equal(t, 4, val)
	})

	t.Run("clear resets capacity", func(t *testing.T) {
		c := NewCache(3)
		c.Set("A", 1)
		c.Set("B", 2)
		c.Set("C", 3)
		c.Clear()

		c.Set("D", 4)
		c.Set("E", 5)
		c.Set("F", 6)

		for _, tc := range []struct {
			key      Key
			val      interface{}
			exists   bool
		}{
			{"D", 4, true},
			{"E", 5, true},
			{"F", 6, true},
			{"A", nil, false},
			{"B", nil, false},
			{"C", nil, false},
		} {
			val, ok := c.Get(tc.key)
			require.Equal(t, tc.exists, ok)
			if tc.exists {
				require.Equal(t, tc.val, val)
			}
		}
	})

	// после Clear() можно снова полностью заполнить кэш до capacity
	// (баг: если не очищать очередь, кэш теряет ёмкость после Clear)
	t.Run("clear then refill to full capacity", func(t *testing.T) {
		c := NewCache(2)
		c.Set("A", 1)
		c.Set("B", 2)
		c.Clear()

		c.Set("A", 1)
		c.Set("B", 2)

		val, ok := c.Get("A")
		require.True(t, ok)
		require.Equal(t, 1, val)

		val, ok = c.Get("B")
		require.True(t, ok)
		require.Equal(t, 2, val)
	})

	// выталкивания давно используемых элементов
	t.Run("lru overflow", func(t *testing.T) {
		c := NewCache(3)
		c.Set("A", 1)
		c.Set("B", 2)
		c.Set("C", 3)
		c.Get("A")
		c.Set("B", 20)
		c.Set("D", 4)

		val, ok := c.Get("C")
		require.False(t, ok)
		require.Nil(t, val)

		val, ok = c.Get("A")
		require.True(t, ok)
		require.Equal(t, 1, val)

		val, ok = c.Get("B")
		require.True(t, ok)
		require.Equal(t, 20, val)

		val, ok = c.Get("D")
		require.True(t, ok)
		require.Equal(t, 4, val)
	})
}

func TestCacheMultithreading(_ *testing.T) {
	c := NewCache(10)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1_000_000; i++ {
			c.Set(Key(strconv.Itoa(i)), i)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1_000_000; i++ {
			c.Get(Key(strconv.Itoa(rand.Intn(1_000_000))))
		}
	}()

	wg.Wait()
}
