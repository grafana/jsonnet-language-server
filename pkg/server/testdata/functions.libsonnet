local myfunc(arg1, arg2) = {
  atb1: arg1,
  atb2: arg2,
};

{
  objFunc(arg1, arg2, arg3): {
    a: arg1,
    b: arg2,
    c: 'hello',
    test: myfunc(arg1, arg2),
  },
}
