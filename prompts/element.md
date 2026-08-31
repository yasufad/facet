
## Order of work, because three packages are moving

`window` implements `element.Frame`. The moment two methods appear on that interface,
`window` stops compiling until it has both. Declaring the interface first and
implementing it second breaks the tree in between, which has happened twice already.

So the order is inside out:

1. **`layout` exports** `OptF32`, `LeafMeasureFunc` and `ComputeLeafLayout`.
   Both other packages need the types. Blocked on nothing.
2. **You declare `MeasureFunc`** and nothing else. A new named type breaks no one,
   and `window` cannot name the callback until it exists. Stop and say so.
3. **`window` implements** both methods and widens the `ShapeLine` phase rule. Extra
   methods on a concrete type satisfy nothing yet and break nothing.
4. **You add both methods to `Frame`**, teach `fakeFrame` about them, and write
   `Text`. `window` already satisfies the enlarged interface, so the tree never
   breaks.

Step 2 is a single small commit. Do not fold it into step 4.

The `text` origin question runs alongside all of this and lands before step 4, since
`Text`'s paint depends on the answer. Produce the evidence; do not change `text`.
