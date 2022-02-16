{
  a: {
    hello: 'this',
    nested1: {
      hello: 'will be',
    },
    nested2: {
      hello: 'clobbered',
    },
  },

}
+ {
  local extensionFromLocal = {
    nested1: {
      this: 'will also be clobbered',
    },
  },
  a+: extensionFromLocal,
}
