# prometheus

grafonnet.query.prometheus

## Index

* [`fn new(datasource, expr)`](#fn-new)
* [`fn withDatasource(value)`](#fn-withdatasource)
* [`fn withEditorMode(value)`](#fn-witheditormode)
* [`fn withExemplar(value=true)`](#fn-withexemplar)
* [`fn withExpr(value)`](#fn-withexpr)
* [`fn withFormat(value)`](#fn-withformat)
* [`fn withHide(value=true)`](#fn-withhide)
* [`fn withInstant(value=true)`](#fn-withinstant)
* [`fn withInterval(value)`](#fn-withinterval)
* [`fn withIntervalFactor(value)`](#fn-withintervalfactor)
* [`fn withLegendFormat(value)`](#fn-withlegendformat)
* [`fn withQueryType(value)`](#fn-withquerytype)
* [`fn withRange(value=true)`](#fn-withrange)
* [`fn withRefId(value)`](#fn-withrefid)
* [`obj datasource`](#obj-datasource)
  * [`fn withType(value)`](#fn-datasourcewithtype)
  * [`fn withUid(value)`](#fn-datasourcewithuid)

## Fields

### fn new

```jsonnet
new(datasource, expr)
```

PARAMETERS:

* **datasource** (`string`)
* **expr** (`string`)

Creates a new prometheus query target for panels.
### fn withDatasource

```jsonnet
withDatasource(value)
```

PARAMETERS:

* **value** (`string`)

Set the datasource for this query.
### fn withEditorMode

```jsonnet
withEditorMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"code"`, `"builder"`

Specifies which editor is being used to prepare the query. It can be "code" or "builder"
### fn withExemplar

```jsonnet
withExemplar(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Execute an additional query to identify interesting raw samples relevant for the given expr
### fn withExpr

```jsonnet
withExpr(value)
```

PARAMETERS:

* **value** (`string`)

The actual expression/query that will be evaluated by Prometheus
### fn withFormat

```jsonnet
withFormat(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"time_series"`, `"table"`, `"heatmap"`

Query format to determine how to display data points in panel. It can be "time_series", "table", "heatmap"
### fn withHide

```jsonnet
withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.
### fn withInstant

```jsonnet
withInstant(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Returns only the latest value that Prometheus has scraped for the requested time series
### fn withInterval

```jsonnet
withInterval(value)
```

PARAMETERS:

* **value** (`string`)

An additional lower limit for the step parameter of the Prometheus query and for the
`$__interval` and `$__rate_interval` variables.
### fn withIntervalFactor

```jsonnet
withIntervalFactor(value)
```

PARAMETERS:

* **value** (`string`)

Set the interval factor for this query.
### fn withLegendFormat

```jsonnet
withLegendFormat(value)
```

PARAMETERS:

* **value** (`string`)

Set the legend format for this query.
### fn withQueryType

```jsonnet
withQueryType(value)
```

PARAMETERS:

* **value** (`string`)

Specify the query flavor
TODO make this required and give it a default
### fn withRange

```jsonnet
withRange(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Returns a Range vector, comprised of a set of time series containing a range of data points over time for each time series
### fn withRefId

```jsonnet
withRefId(value)
```

PARAMETERS:

* **value** (`string`)

A unique identifier for the query within the list of targets.
In server side expressions, the refId is used as a variable name to identify results.
By default, the UI will assign A->Z; however setting meaningful names may be useful.
### obj datasource


#### fn datasource.withType

```jsonnet
datasource.withType(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
#### fn datasource.withUid

```jsonnet
datasource.withUid(value)
```

PARAMETERS:

* **value** (`string`)

Specific datasource instance