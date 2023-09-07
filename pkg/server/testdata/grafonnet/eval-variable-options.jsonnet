local g = import 'github.com/grafana/grafonnet/gen/grafonnet-v10.0.0/main.libsonnet';

g.dashboard.new('title')
+ g.dashboard.withVariables([
  g.dashboard.variable.custom.new('var', ['a'])
  + g.dashboard.variable.custom.selectionOptions.withMulti(),
])
