local somevar = 'hello';

{
  foo: 'bar',
} + {
  local somevar2 = 'world',
  bar: 'foo',
}
