import { useState, useEffect } from 'react';
import './App.css';
import {
    SwitchNodeVersion,
    GetAvailableNodeVersions,
    GetInstalledNodeVersions,
    InstallNodeVersion,
    UninstallNodeVersion // 引入卸载函数
} from '../wailsjs/go/main/App';

// 改进版 compareVersions，确保按主、次、小版本的降序排列
const compareVersions = (a, b) => {
    const parseVersion = (version) => version.split('.').map((num) => parseInt(num, 10));
    const [aMajor, aMinor, aPatch] = parseVersion(a.Version);
    const [bMajor, bMinor, bPatch] = parseVersion(b.Version);

    if (aMajor !== bMajor) return bMajor - aMajor;
    if (aMinor !== bMinor) return bMinor - aMinor;
    return bPatch - aPatch;
};

function App() {
    const [activeTab, setActiveTab] = useState('versions');
    const [availableVersions, setAvailableVersions] = useState([]);
    const [installedVersions, setInstalledVersions] = useState([]);
    const [result, setResult] = useState('');
    const [maxHeight, setMaxHeight] = useState(window.innerHeight - 220);
    const [loadingVersion, setLoadingVersion] = useState(null);
    const [loadingAction, setLoadingAction] = useState(''); // 可以是 'install', 'uninstall', 或 'switch'

    // 动态计算 maxHeight，当窗口大小变化时更新
    useEffect(() => {
        const updateMaxHeight = () => setMaxHeight(window.innerHeight - 220);
        window.addEventListener('resize', updateMaxHeight);
        return () => window.removeEventListener('resize', updateMaxHeight);
    }, []);

    // 获取最新已安装版本列表
    const fetchInstalledVersions = async () => {
        try {
            const installed = await GetInstalledNodeVersions();
            setInstalledVersions(installed.sort(compareVersions));
        } catch (error) {
            setResult('Error fetching installed versions');
        }
    };

    // 获取所有版本信息
    const fetchAvailableVersions = async () => {
        try {
            const available = await GetAvailableNodeVersions();
            setAvailableVersions(available.sort(compareVersions));
        } catch (error) {
            setResult('Error fetching available versions');
        }
    };

    useEffect(() => {
        const fetchVersions = async () => {
            await fetchAvailableVersions();
            await fetchInstalledVersions();
        };
        fetchVersions();
    }, []);

    const handleInstallVersion = async (version) => {
        setLoadingVersion(version);
        setLoadingAction('install');
        try {
            const response = await InstallNodeVersion(version);
            setResult(response);
            await fetchInstalledVersions();
            await fetchAvailableVersions();
        } catch (error) {
            setResult('Error installing version');
        } finally {
            setLoadingVersion(null);
            setLoadingAction('');
        }
    };

    const handleSwitchVersion = async (version) => {
        setLoadingVersion(version);
        setLoadingAction('switch');
        try {
            const response = await SwitchNodeVersion(version);
            setResult(response);
            await fetchInstalledVersions();
        } catch (error) {
            setResult('Error switching version');
        } finally {
            setLoadingVersion(null);
            setLoadingAction('');
        }
    };

    const handleUninstallVersion = async (version) => {
        setLoadingVersion(version);
        setLoadingAction('uninstall');
        try {
            const response = await UninstallNodeVersion(version);
            setResult(response);
            await fetchInstalledVersions();
            await fetchAvailableVersions();
        } catch (error) {
            setResult('Error uninstalling version');
        } finally {
            setLoadingVersion(null);
            setLoadingAction('');
        }
    };

    return (
        <div className="App min-h-screen bg-gray-900 text-white p-6 overflow-hidden">
            <h1 className="text-3xl font-bold mb-6 text-center">Node.js Version Manager</h1>

            <div className="tabs flex justify-center space-x-4 mb-6">
                <button
                    className={`px-4 py-2 ${activeTab === 'versions' ? 'bg-blue-500' : 'bg-gray-700'} rounded`}
                    onClick={() => setActiveTab('versions')}
                    disabled={loadingVersion !== null} // 禁用切换标签按钮
                >
                    Versions
                </button>
                <button
                    className={`px-4 py-2 ${activeTab === 'installed' ? 'bg-blue-500' : 'bg-gray-700'} rounded`}
                    onClick={() => setActiveTab('installed')}
                    disabled={loadingVersion !== null} // 禁用切换标签按钮
                >
                    Installed
                </button>
            </div>

            {activeTab === 'versions' && (
                <div
                    className="version-list bg-gray-800 p-4 rounded-lg custom-scrollbar"
                    style={{ maxHeight: `${maxHeight}px`, overflowY: 'auto' }}
                >
                    <h2 className="text-xl font-semibold mb-4">Available Versions</h2>
                    <table className="table-auto w-full text-left">
                        <thead>
                            <tr>
                                <th className="px-4 py-2">Version</th>
                                <th className="px-4 py-2">Status</th>
                                <th className="px-4 py-2">Operation</th>
                            </tr>
                        </thead>
                        <tbody>
                            {availableVersions.map((versionInfo, index) => (
                                <tr key={index} className="border-t border-gray-700">
                                    <td className="px-4 py-2">{versionInfo.Version}</td>
                                    <td className="px-4 py-2">
                                        {versionInfo.Status === "Installed" ? (
                                            <span className="text-green-500">Installed</span>
                                        ) : (
                                            <span className="text-red-500">Not Installed</span>
                                        )}
                                    </td>
                                    <td className="px-4 py-2">
                                        {versionInfo.Status === "Installed" ? (
                                            <button
                                                className="operation-button bg-gray-600 text-gray-300"
                                                onClick={() => handleSwitchVersion(versionInfo.Version)}
                                                disabled={loadingVersion !== null}
                                            >
                                                {loadingVersion === versionInfo.Version && loadingAction === 'switch' ? (
                                                    <span className="loader"></span>
                                                ) : (
                                                    'Use'
                                                )}
                                            </button>
                                        ) : (
                                            <button
                                                className="operation-button bg-blue-600 text-white"
                                                onClick={() => handleInstallVersion(versionInfo.Version)}
                                                disabled={loadingVersion !== null}
                                            >
                                                {loadingVersion === versionInfo.Version && loadingAction === 'install' ? (
                                                    <span className="loader"></span>
                                                ) : (
                                                    'Install'
                                                )}
                                            </button>
                                        )}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            {activeTab === 'installed' && (
                <div
                    className="installed-list bg-gray-800 p-4 rounded-lg custom-scrollbar"
                    style={{ maxHeight: `${maxHeight}px`, overflowY: 'auto' }}
                >
                    <h2 className="text-xl font-semibold mb-4">Installed Versions</h2>
                    <table className="table-auto w-full text-left">
                        <thead>
                            <tr>
                                <th className="px-4 py-2">Version</th>
                                <th className="px-4 py-2">Operation</th>
                            </tr>
                        </thead>
                        <tbody>
                            {installedVersions.map((versionData, index) => (
                                <tr key={index} className="border-t border-gray-700">
                                    <td className="px-4 py-2">
                                        {versionData.Version}{" "}
                                        {versionData.IsCurrent && (
                                            <span className="text-sm text-blue-400">(Current)</span>
                                        )}
                                    </td>
                                    <td className="px-4 py-2">
                                        <div className="flex space-x-2">
                                            {versionData.IsCurrent ? (
                                                <button
                                                    className="operation-button bg-gray-600 text-gray-300 cursor-not-allowed"
                                                    disabled
                                                >
                                                    Current
                                                </button>
                                            ) : (
                                                <>
                                                    <button
                                                        className="operation-button bg-green-600 text-white"
                                                        onClick={() => handleSwitchVersion(versionData.Version)}
                                                        disabled={loadingVersion !== null}
                                                    >
                                                        {loadingVersion === versionData.Version && loadingAction === 'switch' ? (
                                                            <span className="loader"></span>
                                                        ) : (
                                                            'Switch'
                                                        )}
                                                    </button>
                                                    <button
                                                        className="operation-button bg-red-600 text-white"
                                                        onClick={() => handleUninstallVersion(versionData.Version)}
                                                        disabled={loadingVersion !== null}
                                                    >
                                                        {loadingVersion === versionData.Version && loadingAction === 'uninstall' ? (
                                                            <span className="loader"></span>
                                                        ) : (
                                                            'Uninstall'
                                                        )}
                                                    </button>
                                                </>
                                            )}
                                        </div>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            <p className="text-sm text-gray-400 mt-4">{result}</p>
        </div>
    );
}

export default App;