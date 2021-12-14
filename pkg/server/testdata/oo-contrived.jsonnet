local Base = {
  foo: 2,
  g: self.foo + 100,
};

local WrapperBase = {
  Base: Base,
};

{
  Derived: Base + {
    f: 5,
    old_f: super.foo,
    old_g: super.g,
  },
  WrapperDerived: WrapperBase + {
    Base+: { f: 5 },
  },
}