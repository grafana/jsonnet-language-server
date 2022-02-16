(import 'goto-overrides-base.jsonnet') {
  // 1. Initial definition (overrides everything in the base, except the "a" map)
  a+: {
    hello: 'world',
    nested1: {
      hello: 'world',
    },
    nested2: {
      hello: 'world',
    },
  },
}
+ {
  // 2. Override maps but keep string keys
  a+: {
    hello2: 'world2',
    nested1+: {
      hello2: 'world2',
    },
  },
}
+ {
  // 3. Clobber some attributes
  a+: {
    hello2: 'clobbered',  // Clobber a string
    nested1+: {
      hello: 'clobbered',  // Clobber a nested attribute
    },
    nested2: {},  // Clobber the whole map
  },
}
+ {
  map_overrides: self.a,  // This should refer to all three definitions (initial + 2 overrides)
  nested_map_overrides: self.a.nested1,  // This should refer to all three definitions (initial + 2 overrides)

  carried_string: self.a.hello,  // This should refer to the initial definition (map 1)
  carried_nested_string: self.a.nested1.hello2,  // This should refer to the initial definition (map 2)

  clobbered_string: self.a.hello2,  // This should refer to the override only (map 3)
  clobbered_nested_string: self.a.nested1.hello,  // This should refer to the override only (map 3)
  clobbered_map: self.a.nested2,  // This should refer to the override only (map 3)
}
