(import 'goto-overrides-base.jsonnet') +  // 1. Initial definition from base file
{  // 2. Override nested string
  a+: {
    hello: 'world',
    nested1+: {
      hello: 'world',
    },
    nested2: {
      hello: 'world',
    },
  },
}
+ {
  // 3. Override maps but keep string keys
  a+: {
    hello2: 'world2',
    nested1+: {
      hello2: 'world2',
    },
  },
}
+ {
  // 4. Clobber some attributes
  a+: {
    hello2: 'clobbered',  // Clobber a string
    nested1+: {
      hello: 'clobbered',  // Clobber a nested attribute
    },
    nested2: {},  // Clobber the whole map
  },
}
+ {
  map_overrides: self.a,  // This should refer to all definitions
  nested_map_overrides: self.a.nested1,  // This should refer to all definitions

  carried_string: self.a.hello,  // This should refer to the initial definition (map 2)
  carried_nested_string: self.a.nested1.hello2,  // This should refer to the initial definition (map 3)
  carried_nested_string_from_local: self.a.nested1.from_local,  // This should refer to the definition specified in a local in the base file
  carried_nested_string_from_import: self.a.nested1.from_import,  // This should refer to the definition specified in an import in the base file
  carried_nested_string_from_second_import: self.a.nested1.from_second_import,  // This should refer to the definition specified in an import in the base file

  clobbered_string: self.a.hello2,  // This should refer to the override only (map 4)
  clobbered_nested_string: self.a.nested1.hello,  // This should refer to the override only (map 4)
  clobbered_map: self.a.nested2,  // This should refer to the override only (map 4)
}
