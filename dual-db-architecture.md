Use sqlite for metadata and use bbolt for storing chunks.

Sqlite allows for very flexible data retrival, but it is not fast enough for large texts.
Bbolt is plenty fast, but it is not