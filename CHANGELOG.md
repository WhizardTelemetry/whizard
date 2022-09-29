## 0.5.0-rc.0 / 2022-09-29

This is the first release of whizard.

## What's new

Whizard is a distributed cloud observability platform that provides unified observability (currently monitoring and alerting) for Multi-Cloud, On-Premise and Edge infrastructures and applications. 

### Kubernetes native deployment and management

The Whizard Controller Manager simplifies and automates the configuration and deployment of the whizard components by the following CRDs:  

- `Compactor`: Defines the Compactor component, which does the block compaction and life cycle management for the object storages.
- `Gateway`: Defines the Gateway component, which provides an unified entrypoints for metrics read and write requests. 
- `Ingester`: Defines the Ingester component, which receives metrics data from routers, caches data in memory, flushs data to disk, and finally uploads metrics blocks to object storage.
- `Query`: Defines the Query component, which fetches data from the ingesters and/or stores and then evaluates the query.
- `QueryFrontend`: Defines the Query Frontend component, which improves the query performance by request spliting and result caching.
- `Router`: Defines the Router component, which routes and replicates the metrics to the ingesters. 
- `Ruler`: Defines the Ruler component, which evaluates recording and alerting rules.
- `Service`: Defines a Whizard Service, which connects different whizard components together to provide a complete monitoring service. It also contains shared configurations of different components. 
- `Storage`: Defines the Storage instance, which contains the object storage configuration, and a block manager component for the block inspection and GC.
- `Store`: Defines the Store instance, which facilitates metrics reads from the object storage.
- `Tenant`: Defines a tenant which is the basic unit of resource isolation and auto scaling.  

### Multi-tenancy and Auto scaling

- Whizard components support multi-tenancy and are able to auto scale. 
- The store component supports to auto scale based on its actual load. 
- The ruler component also can also scale based on rule group sharding for a single tenant with too many rules.  

### Data management and GC

- Whizard provides metrics data life cycle management for data on disk or in object storage. If a tenant is deleted, Whizard can automatically cleanup this tenant's blocks in the object storage or on local disk.

### Others

- Whizard also has an agent proxy component that implements the Prometheus HTTP v1 API (reads/writes), which can be used as a data collection agent and a query proxy.

