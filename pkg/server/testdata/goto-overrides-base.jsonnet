{
  a: {
    hello: 'this will be clobbered',
    nested1: {
      hello: 'this will be clobbered',
    },
    nested2: {},
  },

}
+ {
  local extensionFromLocal = {
    nested1+: {
      from_local: 'hey!',
    },
  },
  a+: extensionFromLocal,
}
+ {
  a+: (import 'goto-overrides-imported.jsonnet'),
}
