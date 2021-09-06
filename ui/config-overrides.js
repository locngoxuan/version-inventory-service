const path = require("path");
const {
  override,
  disableEsLint,
  setWebpackOptimizationSplitChunks,
  babelInclude,
  disableChunk,
} = require("customize-cra");

module.exports = function (config, env) {
  console.log(config);
  config.optimization.runtimeChunk = false;
  config.resolve.symlinks = true;
  const modules = ["node_modules"];
  const babelIncludeDir = [
    path.resolve("src"), // don't forget this
    path.resolve("node_modules"),
  ];
  config.resolve.modules = modules;
  return Object.assign(
    config,
    override(
      disableEsLint(),
      setWebpackOptimizationSplitChunks({
        cacheGroups: {
          default: false,
        },
      }),
      disableChunk(),
      babelInclude(babelIncludeDir)
    )(config, env)
  );
};
