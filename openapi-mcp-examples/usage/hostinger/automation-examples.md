# hostinger - Automation Scripts

Complete automation workflows for Hostinger cloud hosting and domain management

**Complexity Level:** expert
**Generated:** 2025-08-30T00:29:20.885Z

## Examples

### Complete Automation Script

Full automation script for hostinger

**Category:** automation

```javascript
#!/usr/bin/env node

/**
 * Hostinger Automation Script
 * 
 * Complete automation workflow for Hostinger cloud hosting and domain management
 */

import { MCPClient } from './mcp-client.js';
import { Logger } from './logger.js';

class HostingerAutomation {
  constructor(config) {
    this.config = config;
    this.client = new MCPClient(config.mcpServer);
    this.logger = new Logger('hostinger-automation');
    this.stats = {
      processed: 0,
      errors: 0,
      startTime: Date.now()
    };
  }

  async run() {
    try {
      this.logger.info('Starting hostinger automation...');
      
      // Initialize MCP connection
      await this.client.initialize();
      
      // Run main workflow
      await this.executeWorkflow();
      
      // Generate report
      await this.generateReport();
      
      this.logger.info('Automation completed successfully');
      
    } catch (error) {
      this.logger.error('Automation failed:', error);
      throw error;
    } finally {
      await this.cleanup();
    }
  }

  async executeWorkflow() {
    const steps = [
      { name: 'discovery', handler: this.discoveryStep.bind(this) },
      { name: 'processing', handler: this.processingStep.bind(this) },
      { name: 'validation', handler: this.validationStep.bind(this) },
      { name: 'cleanup', handler: this.cleanupStep.bind(this) }
    ];

    for (const step of steps) {
      try {
        this.logger.info(`Executing step: ${step.name}`);
        await step.handler();
        this.logger.info(`Step completed: ${step.name}`);
      } catch (error) {
        this.logger.error(`Step failed: ${step.name}`, error);
        if (this.config.stopOnError) {
          throw error;
        }
        this.stats.errors++;
      }
    }
  }

  async discoveryStep() {
    // Discover available resources
    const result = await this.client.call('hostinger.billing_getCatalogItemListV1', {});
    this.discoveredItems = result.data || [];
    this.logger.info(`Discovered ${this.discoveredItems.length} items`);
  }

  async processingStep() {
    // Process each discovered item
    for (const item of this.discoveredItems) {
      try {
        await this.processItem(item);
        this.stats.processed++;
      } catch (error) {
        this.logger.error(`Failed to process item ${item.id}`, error);
        this.stats.errors++;
      }
    }
  }

  async processItem(item) {
    // Example processing logic
    const updateTool = 'hostinger.DNS_updateDNSRecordsV1';
    
    await this.client.call(updateTool, {
      id: item.id,
      ...this.config.updateData
    });
    
    this.logger.debug(`Processed item: ${item.id}`);
  }

  async validationStep() {
    // Validate results
    this.logger.info('Validating automation results...');
    
    const validationResults = await this.client.call('hostinger.billing_getCatalogItemListV1', {
      validate: true
    });
    
    if (!validationResults.success) {
      throw new Error('Validation failed');
    }
  }

  async cleanupStep() {
    // Cleanup temporary resources
    this.logger.info('Cleaning up temporary resources...');
    // Implementation depends on service
  }

  async generateReport() {
    const duration = Date.now() - this.stats.startTime;
    const report = {
      automation: 'hostinger',
      timestamp: new Date().toISOString(),
      duration: `${Math.round(duration / 1000)}s`,
      stats: this.stats,
      success: this.stats.errors === 0
    };

    this.logger.info('Automation Report:', report);
    
    // Save report to file
    await this.saveReport(report);
  }

  async saveReport(report) {
    const fs = await import('fs/promises');
    const reportPath = `reports/hostinger-${Date.now()}.json`;
    await fs.writeFile(reportPath, JSON.stringify(report, null, 2));
    this.logger.info(`Report saved to: ${reportPath}`);
  }

  async cleanup() {
    if (this.client) {
      await this.client.disconnect();
    }
  }
}

// Configuration
const config = {
  mcpServer: {
    command: 'node',
    args: ['examples/mcp-hostinger-server.js'],
    env: {
      HOSTINGER_API_TOKEN: process.env.HOSTINGER_API_TOKEN
    }
  },
  stopOnError: false,
  updateData: {
    // Service-specific update data
  }
};

// Run automation
if (import.meta.url === `file://${process.argv[1]}`) {
  const automation = new HostingerAutomation(config);
  automation.run().catch(console.error);
}

export { HostingerAutomation };
```

### Monitoring & Alerting

Automated monitoring with alerts

**Category:** monitoring

```javascript
// Monitoring and alerting system
class HostingerMonitor {
  constructor(config) {
    this.config = config;
    this.client = new MCPClient(config.mcpServer);
    this.alerts = [];
    this.metrics = {
      requests: 0,
      errors: 0,
      responseTime: []
    };
  }

  async startMonitoring() {
    console.log('Starting hostinger monitoring...');
    
    // Set up periodic health checks
    setInterval(() => this.healthCheck(), this.config.healthCheckInterval || 60000);
    
    // Set up metric collection
    setInterval(() => this.collectMetrics(), this.config.metricsInterval || 30000);
    
    // Set up alert processing
    setInterval(() => this.processAlerts(), this.config.alertInterval || 10000);
  }

  async healthCheck() {
    const startTime = Date.now();
    
    try {
      // Test basic connectivity
      const result = await this.client.call('hostinger.billing_getCatalogItemListV1', {});
      
      const responseTime = Date.now() - startTime;
      this.metrics.responseTime.push(responseTime);
      this.metrics.requests++;
      
      // Check for performance issues
      if (responseTime > this.config.slowResponseThreshold || 5000) {
        this.addAlert('performance', `Slow response: ${responseTime}ms`);
      }
      
      console.log(`Health check passed: ${responseTime}ms`);
      
    } catch (error) {
      this.metrics.errors++;
      this.addAlert('error', `Health check failed: ${error.message}`);
      console.error('Health check failed:', error.message);
    }
  }

  async collectMetrics() {
    const metrics = {
      timestamp: new Date().toISOString(),
      requests: this.metrics.requests,
      errors: this.metrics.errors,
      errorRate: this.metrics.errors / this.metrics.requests,
      avgResponseTime: this.calculateAverageResponseTime(),
      p95ResponseTime: this.calculatePercentile(95)
    };

    // Store metrics
    await this.storeMetrics(metrics);
    
    // Check thresholds
    if (metrics.errorRate > (this.config.errorRateThreshold || 0.1)) {
      this.addAlert('error_rate', `High error rate: ${(metrics.errorRate * 100).toFixed(2)}%`);
    }
  }

  calculateAverageResponseTime() {
    if (this.metrics.responseTime.length === 0) return 0;
    const sum = this.metrics.responseTime.reduce((a, b) => a + b, 0);
    return sum / this.metrics.responseTime.length;
  }

  calculatePercentile(percentile) {
    if (this.metrics.responseTime.length === 0) return 0;
    const sorted = [...this.metrics.responseTime].sort((a, b) => a - b);
    const index = Math.ceil((percentile / 100) * sorted.length) - 1;
    return sorted[index];
  }

  addAlert(type, message) {
    const alert = {
      type,
      message,
      timestamp: new Date().toISOString(),
      service: 'hostinger'
    };
    
    this.alerts.push(alert);
    console.warn(`ALERT [${type}]: ${message}`);
  }

  async processAlerts() {
    if (this.alerts.length === 0) return;
    
    const alertsToProcess = [...this.alerts];
    this.alerts = [];
    
    for (const alert of alertsToProcess) {
      await this.sendAlert(alert);
    }
  }

  async sendAlert(alert) {
    // Send to configured alert channels
    const channels = this.config.alertChannels || ['console'];
    
    for (const channel of channels) {
      switch (channel) {
        case 'console':
          console.error('ALERT:', alert);
          break;
        case 'webhook':
          await this.sendWebhookAlert(alert);
          break;
        case 'email':
          await this.sendEmailAlert(alert);
          break;
      }
    }
  }

  async storeMetrics(metrics) {
    // Store in configured storage
    const storage = this.config.metricsStorage || 'file';
    
    switch (storage) {
      case 'file':
        const fs = await import('fs/promises');
        await fs.appendFile('metrics/hostinger.jsonl', JSON.stringify(metrics) + '\n');
        break;
      case 'database':
        // Store in database
        break;
    }
  }
}

// Usage
const monitor = new HostingerMonitor({
  mcpServer: { /* MCP server config */ },
  healthCheckInterval: 60000,
  metricsInterval: 30000,
  alertInterval: 10000,
  slowResponseThreshold: 5000,
  errorRateThreshold: 0.1,
  alertChannels: ['console', 'webhook'],
  metricsStorage: 'file'
});

monitor.startMonitoring();
```

### Scheduled Tasks

Setting up scheduled automation tasks

**Category:** scheduling

```javascript
// Scheduled task system
import cron from 'node-cron';

class HostingerScheduler {
  constructor(config) {
    this.config = config;
    this.client = new MCPClient(config.mcpServer);
    this.tasks = new Map();
    this.running = false;
  }

  async start() {
    console.log('Starting hostinger scheduler...');
    this.running = true;
    
    // Initialize MCP client
    await this.client.initialize();
    
    // Set up scheduled tasks
    this.setupTasks();
    
    console.log(`Scheduler started with ${this.tasks.size} tasks`);
  }

  setupTasks() {
    const tasks = [
      {
        name: 'daily-sync',
        schedule: '0 2 * * *', // Daily at 2 AM
        handler: this.dailySync.bind(this)
      },
      {
        name: 'hourly-check',
        schedule: '0 * * * *', // Every hour
        handler: this.hourlyCheck.bind(this)
      },
      {
        name: 'weekly-cleanup',
        schedule: '0 3 * * 0', // Weekly on Sunday at 3 AM
        handler: this.weeklyCleanup.bind(this)
      }
    ];

    for (const task of tasks) {
      const cronTask = cron.schedule(task.schedule, async () => {
        if (!this.running) return;
        
        console.log(`Running scheduled task: ${task.name}`);
        
        try {
          await task.handler();
          console.log(`Task completed: ${task.name}`);
        } catch (error) {
          console.error(`Task failed: ${task.name}`, error);
          await this.handleTaskError(task.name, error);
        }
      }, {
        scheduled: false,
        timezone: this.config.timezone || 'UTC'
      });

      this.tasks.set(task.name, {
        ...task,
        cronTask,
        lastRun: null,
        runCount: 0,
        errorCount: 0
      });

      cronTask.start();
    }
  }

  async dailySync() {
    // Daily synchronization task
    console.log('Executing daily sync...');
    
    const items = await this.client.call('hostinger.billing_getCatalogItemListV1', {});
    
    for (const item of items.data || []) {
      await this.syncItem(item);
    }
    
    console.log(`Daily sync completed: ${items.data?.length || 0} items processed`);
  }

  async hourlyCheck() {
    // Hourly health check
    console.log('Executing hourly check...');
    
    const status = await this.client.call('hostinger.billing_getCatalogItemListV1', {
      healthCheck: true
    });
    
    if (!status.healthy) {
      await this.handleUnhealthyStatus(status);
    }
  }

  async weeklyCleanup() {
    // Weekly cleanup task
    console.log('Executing weekly cleanup...');
    
    // Clean up old data
    const cutoffDate = new Date();
    cutoffDate.setDate(cutoffDate.getDate() - 30); // 30 days ago
    
    const cleanupResult = await this.client.call('hostinger.cleanup', {
      before: cutoffDate.toISOString()
    });
    
    console.log(`Weekly cleanup completed: ${cleanupResult.deletedCount} items removed`);
  }

  async syncItem(item) {
    // Sync individual item
    try {
      await this.client.call('hostinger.DNS_updateDNSRecordsV1', {
        id: item.id,
        sync: true
      });
    } catch (error) {
      console.error(`Failed to sync item ${item.id}:`, error.message);
    }
  }

  async handleUnhealthyStatus(status) {
    console.warn('Service is unhealthy:', status);
    
    // Send alert
    await this.sendAlert({
      type: 'health',
      message: `Service unhealthy: ${status.message}`,
      severity: 'warning'
    });
  }

  async handleTaskError(taskName, error) {
    const task = this.tasks.get(taskName);
    if (task) {
      task.errorCount++;
    }
    
    // Send error alert
    await this.sendAlert({
      type: 'task_error',
      message: `Task ${taskName} failed: ${error.message}`,
      severity: 'error'
    });
  }

  async sendAlert(alert) {
    // Implementation depends on alert configuration
    console.error('ALERT:', alert);
  }

  getTaskStatus() {
    const status = {};
    for (const [name, task] of this.tasks) {
      status[name] = {
        schedule: task.schedule,
        lastRun: task.lastRun,
        runCount: task.runCount,
        errorCount: task.errorCount,
        nextRun: task.cronTask.nextDate()
      };
    }
    return status;
  }

  async stop() {
    console.log('Stopping scheduler...');
    this.running = false;
    
    for (const [name, task] of this.tasks) {
      task.cronTask.stop();
    }
    
    await this.client.disconnect();
    console.log('Scheduler stopped');
  }
}

// Usage
const scheduler = new HostingerScheduler({
  mcpServer: {
    command: 'node',
    args: ['examples/mcp-hostinger-server.js'],
    env: {
      HOSTINGER_API_TOKEN: process.env.HOSTINGER_API_TOKEN
    }
  },
  timezone: 'UTC'
});

// Start scheduler
scheduler.start().catch(console.error);

// Graceful shutdown
process.on('SIGINT', async () => {
  console.log('Received SIGINT, shutting down gracefully...');
  await scheduler.stop();
  process.exit(0);
});

export { HostingerScheduler };
```

