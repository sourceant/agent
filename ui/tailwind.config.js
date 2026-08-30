import design from '@sourceant/design/tailwind'

/* The design comes from the package. What gets scanned for class names is this
 * application's own question, and it includes the package, whose components
 * carry classes of their own.
 */
export default {
  presets: [design],
  content: [
    './index.html',
    './src/**/*.{js,ts,vue}',
    './node_modules/@sourceant/design/src/**/*.{js,ts,vue}',
  ],
}
