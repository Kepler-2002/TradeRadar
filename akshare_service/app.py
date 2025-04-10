import akshare as ak
import pandas as pd
from flask import Flask, jsonify, request
import requests
import time
import logging
import sys
import random

app = Flask(__name__)

# 配置日志
logging.basicConfig(level=logging.INFO, 
                   format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
                   stream=sys.stdout)
logger = logging.getLogger(__name__)

# AKShare Docker API地址
AKSHARE_API = "http://localhost:5000"

@app.route('/health', methods=['GET'])
def health_check():
    """健康检查接口"""
    logger.info("收到健康检查请求")
    return jsonify({"status": "ok"})

@app.route('/', methods=['GET'])
def root():
    """根路径响应"""
    return jsonify({"message": "AKShare服务正在运行", "status": "ok"})

def generate_mock_quote(symbol):
    """生成模拟行情数据"""
    base_price = random.uniform(10, 100)
    return {
        "symbol": symbol,
        "name": f"模拟_{symbol}",
        "price": base_price,
        "open": base_price * 0.99,
        "high": base_price * 1.02,
        "low": base_price * 0.98,
        "volume": random.randint(100000, 10000000),
        "change_percent": random.uniform(-5, 5),
        "timestamp": int(time.time())
    }

@app.route('/api/quotes', methods=['GET'])
def get_quotes():
    """获取实时行情数据"""
    try:
        # 获取请求参数
        symbols = request.args.get('symbols', '')
        if not symbols:
            return jsonify({"error": "请提供股票代码"}), 400
        
        symbol_list = symbols.split(',')
        logger.info(f"获取行情数据: {symbol_list}")
        
        # 尝试从AKShare Docker获取数据
        try:
            result = []
            for symbol in symbol_list:
                # 根据股票代码格式判断是A股还是港股
                if symbol.endswith('.SH') or symbol.endswith('.SZ'):
                    # A股
                    code = symbol.split('.')[0]
                    
                    # 调用AKShare Docker API获取A股实时行情
                    api_url = f"{AKSHARE_API}/api/public/stock_zh_a_spot_em"
                    response = requests.get(api_url, timeout=10)
                    
                    if response.status_code == 200:
                        data = response.json()
                        # 查找对应的股票
                        stock_data = None
                        for item in data:
                            if item.get('代码') == code:
                                stock_data = item
                                break
                        
                        if stock_data:
                            quote = {
                                "symbol": symbol,
                                "name": stock_data.get('名称', f"未知_{symbol}"),
                                "price": float(stock_data.get('最新价', 0)),
                                "open": float(stock_data.get('开盘价', 0)),
                                "high": float(stock_data.get('最高价', 0)),
                                "low": float(stock_data.get('最低价', 0)),
                                "volume": float(stock_data.get('成交量', 0)),
                                "change_percent": float(stock_data.get('涨跌幅', 0)),
                                "timestamp": int(time.time())
                            }
                            result.append(quote)
                        else:
                            logger.warning(f"未找到股票: {symbol}")
                            result.append(generate_mock_quote(symbol))
                    else:
                        logger.error(f"API请求失败: {response.status_code}")
                        result.append(generate_mock_quote(symbol))
                
                elif symbol.endswith('.HK'):
                    # 港股
                    code = symbol.split('.')[0]
                    
                    # 调用AKShare Docker API获取港股实时行情
                    api_url = f"{AKSHARE_API}/api/public/stock_hk_spot_em"
                    response = requests.get(api_url, timeout=10)
                    
                    if response.status_code == 200:
                        data = response.json()
                        # 查找对应的股票
                        stock_data = None
                        for item in data:
                            if item.get('代码') == code:
                                stock_data = item
                                break
                        
                        if stock_data:
                            quote = {
                                "symbol": symbol,
                                "name": stock_data.get('名称', f"未知_{symbol}"),
                                "price": float(stock_data.get('最新价', 0)),
                                "open": float(stock_data.get('开盘价', 0)),
                                "high": float(stock_data.get('最高价', 0)),
                                "low": float(stock_data.get('最低价', 0)),
                                "volume": float(stock_data.get('成交量', 0)),
                                "change_percent": float(stock_data.get('涨跌幅', 0)),
                                "timestamp": int(time.time())
                            }
                            result.append(quote)
                        else:
                            logger.warning(f"未找到股票: {symbol}")
                            result.append(generate_mock_quote(symbol))
                    else:
                        logger.error(f"API请求失败: {response.status_code}")
                        result.append(generate_mock_quote(symbol))
                else:
                    # 未知类型，使用模拟数据
                    result.append(generate_mock_quote(symbol))
            
            return jsonify(result)
        except Exception as e:
            logger.error(f"调用AKShare Docker API失败: {e}")
            # 使用模拟数据
            result = [generate_mock_quote(symbol) for symbol in symbol_list]
            return jsonify(result)
    
    except Exception as e:
        logger.error(f"处理请求失败: {e}")
        # 发生错误时返回模拟数据
        result = []
        for symbol in symbols.split(','):
            quote = generate_mock_quote(symbol)
            result.append(quote)
        return jsonify(result)

if __name__ == '__main__':
    logger.info("AKShare服务启动中...")
    app.run(host='0.0.0.0', port=5001, debug=False)  # 使用5001端口，避免与AKShare Docker冲突