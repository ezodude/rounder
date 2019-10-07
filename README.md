# Rounder API

Rounder is a POC API for sentiment and topic summaries of a specified subject.

Data for a subject is ingested from multiple data (typically news) providers.

But as a proof concept, __for now Rounder is hardcoded internally for one provider.__ This will change soon.

## How it works

### Ingest endpoint ✅

First you call the `/ingest` endpoint to ensure data for a subject is made available from a data supplier.

The following `ENVs` are required to make this happen:
- `DATA_ENDPOINT` - a url pointing to actual `JSON` data. Should include `_KEY_` & `_SUBJECT_` placeholders.
- `PROVIDER_KEY` - if available, an API Key supplied by the data provider.

For e.g.,

`DATA_ENDPOINT` with placeholders: `http://acmeprovider.com/api/v1/search?key=_KEY_&query=_SUBJECT_%20AND%20sourceCountry:%22United%20Kingdom%22&limit=100&format=json`

### Digest enpoints 🔜

Then, call either `topics/digest` or `sentiment/digest` for a subject's respective summaries.

Both endpoints will return article data aggregated by either topics or sentiment.
