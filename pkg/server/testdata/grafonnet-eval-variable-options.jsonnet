local g = import 'vendor/grafonnet-latest/main.libsonnet';

g.dashboard.new('title')
+ g.dashboard.withVariables([
  g.dashboard.variable.custom.new('var', ['a'])
  + g.dashboard.variable.custom.selectionOptions.withMulti(),
])
