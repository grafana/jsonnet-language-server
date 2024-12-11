{
  util:: {
    new():: {
      local this = self,

      attr: 'unset1',
      attr2: 'unset2',

      withAttr(v):: self {  // Intentionally using `self` instead of `this`
        attr: v,
      },

      withAttr2(v):: this {  // Intentionally using `this` instead of `self`
        attr2: v,
      },

      build():: '%s + %s' % [self.attr, this.attr2],
    },
  },


  test: self.util.new().withAttr('hello').withAttr2('world').build(),
}
