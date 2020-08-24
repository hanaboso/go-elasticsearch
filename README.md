# Hanaboso GO ElasticSearch

**Download**
```
go mod download github.com/hanaboso/go-elasticsearch
```

**Usage**
```
import "github.com/hanaboso/go-elasticsearch"

elasticsearch := &elasticsearch.Connection{}
elasticsearch.Connect("http://elasticseach:9200")

elasticsearch.Client.Search(...)
```
