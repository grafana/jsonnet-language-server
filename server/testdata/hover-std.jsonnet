{
  thisFile: std.thisFile,
  fields: std.objectFields({ test: 'test' }),
  myFunc(): {
    spec+: {
      templates: std.map(function(v) if std.objectHas(v, 'container') then v { container: {
        image: 'alpine:3.13',
        command: ['echo', "Would've run the following container: %s" % std.manifestJson(v.container)],
      } } else v, super.templates),
    },
  },
  listComprehension: [
    item
    for item
    in std.objectFields({ test: 'test' })
    if std.map({ test: 'test' }, item)
  ],
}
