{
  mapComprehension: {
    [item]: item
    for item
    in std.objectFields({ test: 'test' })
    if std.map({ test: 'test' }, item)
  },
}
