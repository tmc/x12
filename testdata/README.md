# testdata

The `.edi` files are the canonical fixture set for this package's
tests. They are example transactions published by [X12](https://x12.org/examples)
for the 005010 implementation guides, fetched in March 2023 with the
crawl tooling in this directory (`Makefile`, `fetch-examples.sh`,
`urls.txt`).

The checked-in files are authoritative: tests read them from disk and
never fetch the network. Refreshing the set (`make testdata-x12`) is a
manual, deliberate change — review the resulting diff before committing
it.

Note that several of the 005010X221 examples pad ISA elements with
trailing whitespace, including the segment ID itself; the round-trip
test decodes those with `WithRelaxedSegmentIDWhitespace`.

The examples are reproduced here for interoperability testing.
X12 publishes them publicly, but redistribution terms have not been
formally confirmed; if X12's terms require it, this set may need to be
replaced with synthetic equivalents.
