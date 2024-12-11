local obj = {
  foo: {
    bar: 'innerfoo',
  },
  bar: 'foo',
};

{
  attr: obj.foo,
  s: self.attr.bar,
}
