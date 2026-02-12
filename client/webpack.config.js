// Webpack打包配置
const path = require('path');
const CopyWebpackPlugin = require('copy-webpack-plugin');

module.exports = {
  // 入口文件
  entry: {
    background: './src/background/background.ts',
    popup: './src/popup/popup.ts',
  },
  
  // 输出配置
  output: {
    path: path.resolve(__dirname, 'dist'),
    filename: '[name].js',
    clean: true, // 自动清理输出目录
  },
  
  // 模块解析
  resolve: {
    extensions: ['.ts', '.js'],
  },
  
  // 模块规则
  module: {
    rules: [
      // TypeScript处理
      {
        test: /\.ts$/,
        use: 'ts-loader',
        exclude: /node_modules/,
      },
      // CSS处理
      {
        test: /\.css$/,
        use: ['style-loader', 'css-loader'],
      },
    ],
  },
  
  // 插件配置
  plugins: [
    // 复制manifest.json到dist目录
    new CopyWebpackPlugin({
      patterns: [
        {
          from: 'public/manifest.json',
          to: 'manifest.json',
        },
        // 复制popup.html到dist目录
        {
          from: 'src/popup/popup.html',
          to: 'popup.html',
        },
      ],
    }),
  ],
  
  // 开发工具
  devtool: 'source-map',
  
  // 模式
  mode: process.env.NODE_ENV === 'production' ? 'production' : 'development',
};
