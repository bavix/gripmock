# Catalog

Shows `output.template`: the response structure is computed from the request, not
filled into a fixed shape.

- `Search` filters the catalog by stock, applies `offset`/`limit`, sums the page
  and counts categories into a `map<string,int32>` whose keys come from the data.
- `WatchStock` emits one message per requested sku per tick, so the number of
  stream messages is the product of two request fields.

```bash
gripmock examples/projects/catalog/service.proto --stub examples/projects/catalog
grpctestify examples/projects/catalog
```
