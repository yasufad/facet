// Ported from Taffy src/tree/cache.rs (MIT).
//
// Per-node cache of layout results. Taffy packs the cache key into u64s by
// reusing the sign bits of f32s to encode the requested axis; Go keeps the same
// packing so the cache hit/miss behaviour is identical. The cache uses a small
// fixed-size array of measure entries with a second-chance replacement cursor.
package layout

import "math"

const cacheSize = 9

var (
	infinityBits    uint32 = math.Float32bits(float32(math.Inf(1)))
	negInfinityBits uint32 = math.Float32bits(float32(math.Inf(-1)))
)

const (
	signBit1         uint64 = 1 << 63
	signBit2         uint64 = 1 << 31
	bothSignBitsMask uint64 = signBit1 | signBit2
	nonSignBitsMask  uint64 = ^bothSignBitsMask
	xAxisValueMask   uint64 = uint64(^uint32(0)) << 32
)

func optionCacheKey(o optF32) uint32 {
	if o.isSome() {
		return math.Float32bits(o.v)
	}
	return infinityBits
}

func sizeOptionCacheKey(s Size[optF32]) uint64 {
	return uint64(optionCacheKey(s.Width))<<32 | uint64(optionCacheKey(s.Height))
}

func availableSpaceCacheKey(a AvailableSpace) uint32 {
	switch a.kind {
	case availableDefinite:
		return math.Float32bits(-a.val)
	case availableMinContent:
		return negInfinityBits
	default:
		return infinityBits
	}
}

func mixedCacheKey(kd optF32, avs AvailableSpace) uint32 {
	if kd.isSome() {
		return math.Float32bits(kd.v)
	}
	return availableSpaceCacheKey(avs)
}

func sizeMixedCacheKey(kd Size[optF32], avs Size[AvailableSpace]) uint64 {
	return uint64(mixedCacheKey(kd.Width, avs.Width))<<32 | uint64(mixedCacheKey(kd.Height, avs.Height))
}

// cacheKey is the space-optimised key for a cache entry.
type cacheKey struct {
	kdAvailableSpace           uint64
	parentSize                 uint64
	knownDimensionsAreDefinite Size[bool]
}

func cacheKeyFromInput(in *LayoutInput) cacheKey {
	var extraBits uint64
	switch in.Axis {
	case requestedHorizontal:
		extraBits = signBit1
	case requestedVertical:
		extraBits = signBit2
	default:
		extraBits = signBit1 | signBit2
	}
	return cacheKey{
		kdAvailableSpace: sizeMixedCacheKey(in.KnownDimensions, in.AvailableSpace),
		parentSize:       (sizeOptionCacheKey(in.ParentSize) & nonSignBitsMask) | extraBits,
		knownDimensionsAreDefinite: sizeZipMap(in.KnownDimensionsAreDefinite, in.KnownDimensions,
			func(isDefinite bool, kd optF32) bool { return isDefinite || !kd.isSome() }),
	}
}

func (k cacheKey) xAxisParentSize() uint64 {
	return k.parentSize & (xAxisValueMask & nonSignBitsMask)
}

func (k cacheKey) requestedAxisBits() uint64 {
	return k.parentSize & bothSignBitsMask
}

// sizeIsValidFor reports whether a cached entry with this key contains a valid
// size for the axis requested by other. Sizes computed for a single axis may
// contain garbage values in the other axis, so an entry is only usable if it
// was computed for the same axis (or for both axes).
func (k cacheKey) sizeIsValidFor(other cacheKey) bool {
	entryAxis := k.requestedAxisBits()
	return entryAxis == bothSignBitsMask || entryAxis == other.requestedAxisBits()
}

// cacheEntry is a single cached result.
type cacheEntry[T any] struct {
	key     cacheKey
	content T
}

// Cache is a per-node cache of layout results.
type Cache struct {
	finalLayoutEntry    *cacheEntry[LayoutOutput]
	measureEntries      [cacheSize]*cacheEntry[Size[float32]]
	recentlyUsedEntries uint16
	nextMeasureEntry    uint8
	isEmpty             bool
}

// NewCache creates a new empty cache.
func NewCache() Cache {
	return Cache{isEmpty: true}
}

// Get tries to retrieve a cached result.
func (c *Cache) Get(in *LayoutInput) (LayoutOutput, bool) {
	key := cacheKeyFromInput(in)
	switch in.RunMode {
	case runPerformLayout:
		if c.finalLayoutEntry != nil && c.finalLayoutEntry.key == key {
			return c.finalLayoutEntry.content, true
		}
		return LayoutOutput{}, false
	case runComputeSize:
		for index, entry := range c.measureEntries {
			if entry == nil {
				continue
			}
			if entry.key.kdAvailableSpace == key.kdAvailableSpace &&
				entry.key.knownDimensionsAreDefinite == key.knownDimensionsAreDefinite &&
				entry.key.xAxisParentSize() == key.xAxisParentSize() &&
				entry.key.sizeIsValidFor(key) {
				c.recentlyUsedEntries |= 1 << index
				return layoutOutputFromOuterSize(entry.content), true
			}
		}
		return LayoutOutput{}, false
	default:
		return LayoutOutput{}, false
	}
}

// Store records a computed result.
func (c *Cache) Store(in *LayoutInput, out LayoutOutput) {
	key := cacheKeyFromInput(in)
	switch in.RunMode {
	case runPerformLayout:
		c.isEmpty = false
		c.finalLayoutEntry = &cacheEntry[LayoutOutput]{key: key, content: out}
	case runComputeSize:
		// Measure entries only store the size, and cache hits are reconstructed
		// with from_outer_size, which resets the margin-collapse metadata. Results
		// that carry such metadata cannot be reconstructed from their size, so
		// don't cache them.
		if out.MarginsCanCollapseThrough ||
			out.TopMargin != collapsibleMarginZero ||
			out.BottomMargin != collapsibleMarginZero {
			return
		}
		c.isEmpty = false
		for index, entry := range c.measureEntries {
			if entry != nil && entry.key == key {
				entry.content = out.Size
				c.recentlyUsedEntries |= 1 << index
				return
			}
		}
		for c.recentlyUsedEntries&(1<<c.nextMeasureEntry) != 0 {
			c.recentlyUsedEntries &= ^(1 << c.nextMeasureEntry)
			c.nextMeasureEntry++
			if c.nextMeasureEntry == cacheSize {
				c.nextMeasureEntry = 0
			}
		}
		entryIndex := int(c.nextMeasureEntry)
		c.measureEntries[entryIndex] = &cacheEntry[Size[float32]]{key: key, content: out.Size}
		c.recentlyUsedEntries |= 1 << entryIndex
		c.nextMeasureEntry++
		if c.nextMeasureEntry == cacheSize {
			c.nextMeasureEntry = 0
		}
	default:
		// PerformHiddenLayout: nothing to cache.
	}
}

// ClearState reports the outcome of a clear operation.
type clearState uint8

const (
	clearStateCleared clearState = iota
	clearStateAlreadyEmpty
)

// Clear empties the cache and reports whether anything was cleared.
func (c *Cache) Clear() clearState {
	if c.isEmpty {
		return clearStateAlreadyEmpty
	}
	c.isEmpty = true
	c.finalLayoutEntry = nil
	for i := range c.measureEntries {
		c.measureEntries[i] = nil
	}
	c.recentlyUsedEntries = 0
	c.nextMeasureEntry = 0
	return clearStateCleared
}

// IsEmpty reports whether all cache entries are nil.
func (c *Cache) IsEmpty() bool {
	if c.finalLayoutEntry != nil {
		return false
	}
	for _, e := range c.measureEntries {
		if e != nil {
			return false
		}
	}
	return true
}
