/** The ids of the catalog jobs: 1 Snowflake at Capgemini, 2 DevOps at Bitdefender, 3 Amazon, 4 Carpathia. */
export const seedId = (n: 1 | 2 | 3 | 4) => `00000000-0000-4000-8000-00000000000${n}`

/** The catalog's four sample jobs as the jobs service returns them (docs/api/jobs.md, "Catalog seed"). */
export const CATALOG_JOBS = [
  {
    id: "00000000-0000-4000-8000-000000000001",
    title: "Snowflake Administrator DevOps",
    company: "Capgemini Romania",
    city: "Iași",
    country: "Romania",
    work_mode: "onsite",
    level: "senior",
    years_required: 5,
    must_have: [
      "Snowflake",
      "SQL",
      "Terraform",
      "Infrastructure as Code",
      "Git",
      "CI/CD",
      "RBAC",
      "Monitoring",
      "Python"
    ],
    nice_to_have: [],
    responsibilities: [
      "Design and administer Snowflake databases, schemas, roles, warehouses, and access-control models supporting Knowledge Domains.",
      "Automate provisioning, configuration, and deployment processes through Infrastructure as Code and CI/CD pipelines.",
      "Implement and maintain governance, security, naming, tagging, and lifecycle-management standards.",
      "Develop and maintain Terraform modules and reusable deployment templates.",
      "Monitor platform usage, performance, reliability, security, and cost consumption.",
      "Troubleshoot technical issues and provide expert guidance on Snowflake capabilities and best practices.",
      "Support onboarding activities for new Knowledge Domains and platform consumers.",
      "Create operational documentation, runbooks, and knowledge-base content.",
      "Collaborate with cross-functional teams in Agile and international delivery environments."
    ],
    about: "As a Snowflake Administrator / DevOps Engineer, you will play a key role in building, automating, and operating the Snowflake foundations supporting an enterprise-scale Knowledge Platform. You will be responsible for enabling secure, scalable, and governed Knowledge Domains while ensuring platform reliability, operational excellence, and adherence to data governance standards. Working alongside architects, engineers, and business stakeholders, you will contribute to the standardization and industrialization of Snowflake platform operations.",
    about_company: "Capgemini is an AI-powered global business and technology transformation partner, delivering tangible business value. We imagine the future of organizations and make it real with AI, technology and people. With our strong heritage of nearly 60 years, we are a responsible and diverse group of 420,000 team members in more than 50 countries. We deliver end-to-end services and solutions with our deep industry expertise and strong partner ecosystem, leveraging our capabilities across strategy, technology, design, engineering and business operations. The Group reported 2024 global revenues of €22.1 billion.",
    product: "enterprise Knowledge Platform on Snowflake",
    perks: [
      "Diverse & Impactful Projects: Develop your expertise across a wide range of missions, technologies, and sectors, with meaningful autonomy and room for initiative.",
      "Continuous Learning: Access comprehensive training platforms and recognized certifications.",
      "Flexibility & Balance: Benefit from hybrid working policies that allow you to manage your time and work location effectively."
    ],
    areas: [
      "DevOps",
      "Data"
    ],
    domains: ["it"],
    url: "https://www.linkedin.com/jobs/view/4300000001",
    source: "catalog",
    read_from: "",
    sample: false,
    added_by_me: false,
    created_at: "2026-11-21T08:00:00Z"
  },
  {
    id: "00000000-0000-4000-8000-000000000002",
    title: "DevOps Engineer",
    company: "Bitdefender",
    city: "Bucharest",
    country: "Romania",
    work_mode: "hybrid",
    level: "mid",
    years_required: 3,
    must_have: [
      "Kubernetes",
      "Docker",
      "CI/CD",
      "Linux",
      "Terraform",
      "AWS",
      "Python",
      "Monitoring"
    ],
    nice_to_have: [],
    responsibilities: [
      "Design and maintain CI/CD pipelines for microservices.",
      "Automate infrastructure provisioning with Terraform and Ansible.",
      "Run and improve Kubernetes clusters on AWS.",
      "Build monitoring, alerting and dashboards for production services.",
      "Respond to incidents and lead blameless post-mortems.",
      "Harden systems and apply security best practices across environments.",
      "Write scripts in Python and Bash to remove manual work.",
      "Create runbooks and documentation for the on-call team.",
      "Collaborate with developers and security engineers in an Agile setup."
    ],
    about: "As a DevOps Engineer, you will help build and run the cloud infrastructure behind security services used by customers around the world. You will own deployment pipelines, improve reliability, and work with developers so that new features reach production safely and quickly. You will join a platform team that values automation, clear documentation, and calm incident response.",
    about_company: "Bitdefender builds cybersecurity products and services. This is a sample listing written for the prototype, so the details are illustrative and it is not an actual opening.",
    product: "cybersecurity services",
    perks: [
      "Hybrid working: split your week between the office and home.",
      "Learning budget: courses, books, and certifications.",
      "Challenging problems: work on security services that people rely on."
    ],
    areas: [
      "DevOps"
    ],
    domains: ["it"],
    url: "",
    source: "catalog",
    read_from: "",
    sample: true,
    added_by_me: false,
    created_at: "2026-11-22T08:00:00Z"
  },
  {
    id: "00000000-0000-4000-8000-000000000003",
    title: "Software Engineer",
    company: "Amazon",
    city: "Bucharest",
    country: "Romania",
    work_mode: "hybrid",
    level: "mid",
    years_required: 3,
    must_have: [
      "Java",
      "Python",
      "Algorithms",
      "Distributed systems",
      "SQL",
      "Git",
      "Testing",
      "AWS"
    ],
    nice_to_have: [],
    responsibilities: [
      "Design and build scalable, distributed services.",
      "Write clean, well-tested code in Java or Python.",
      "Own features from design review to production.",
      "Participate in code reviews and design discussions.",
      "Monitor your services and fix operational issues quickly.",
      "Analyze and improve the performance of existing systems.",
      "Work with product managers to turn requirements into technical designs.",
      "Write technical documentation for the systems the team owns.",
      "Mentor junior engineers and share what you learn."
    ],
    about: "As a Software Engineer, you will design, build, and operate services that customers depend on every day. You will take ownership of features from design to production, write clean and well-tested code, and work with product managers and other engineers to solve ambiguous problems. You will join a team that values ownership, operational excellence, and continuous learning.",
    about_company: "Amazon is a large technology company. This is a sample listing written for the prototype, so the details are illustrative and it is not an actual opening.",
    product: "services that customers depend on every day",
    perks: [
      "Ownership: take real responsibility for what you build.",
      "Growth: structured mentoring and clear career paths.",
      "Flexibility: hybrid work and relocation support."
    ],
    areas: [
      "Backend"
    ],
    domains: ["it"],
    url: "",
    source: "catalog",
    read_from: "",
    sample: true,
    added_by_me: false,
    created_at: "2026-11-23T08:00:00Z"
  },
  {
    id: "00000000-0000-4000-8000-000000000004",
    title: "System Administrator",
    company: "Carpathia Systems",
    city: "Cluj-Napoca",
    country: "Romania",
    work_mode: "onsite",
    level: "mid",
    years_required: 3,
    must_have: [
      "Linux",
      "Windows Server",
      "Active Directory",
      "Networking",
      "Backup",
      "Monitoring",
      "Virtualization",
      "Bash"
    ],
    nice_to_have: [],
    responsibilities: [
      "Install and maintain Linux and Windows servers.",
      "Manage user accounts, permissions, and Active Directory.",
      "Monitor systems and respond to alerts.",
      "Plan and test backups and disaster recovery procedures.",
      "Apply security patches and updates on schedule.",
      "Manage virtualization platforms and storage.",
      "Automate routine tasks with Bash and PowerShell.",
      "Document procedures and keep the asset inventory up to date.",
      "Support colleagues and escalate complex issues to the right teams."
    ],
    about: "As a System Administrator, you will keep the company’s servers, networks, and user environments secure, available, and well documented. You will be the first point of contact for infrastructure issues, plan upgrades and backups, and work with the helpdesk and engineering teams to improve how IT is run day to day.",
    about_company: "Carpathia Systems is a fictional company created for this prototype. It stands in for a mid-size IT services firm, so this listing is a sample and not an actual opening.",
    product: "IT services",
    perks: [
      "A stable team and a predictable on-call rotation.",
      "Training and certifications paid by the company.",
      "Modern office with parking."
    ],
    areas: [
      "Sysadmin"
    ],
    domains: ["it"],
    url: "",
    source: "catalog",
    read_from: "",
    sample: true,
    added_by_me: false,
    created_at: "2026-11-24T08:00:00Z"
  }
]
