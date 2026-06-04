CREATE TABLE clause_library (
    id           text PRIMARY KEY,
    clause_type  text NOT NULL,
    standard_text text NOT NULL,
    notes        text NOT NULL DEFAULT ''
);

INSERT INTO clause_library (id, clause_type, standard_text, notes) VALUES
(
    'lib-liability-001',
    'liability',
    'Each party''s total liability to the other under or in connection with this Agreement, whether arising in contract, tort (including negligence), breach of statutory duty, or otherwise, shall not exceed the total fees paid or payable by the Customer in the twelve (12) months immediately preceding the event giving rise to the claim.',
    'Standard mutual liability cap tied to contract value; preferred over uncapped exposure.'
),
(
    'lib-indemnity-001',
    'indemnity',
    'Each party (the "Indemnifying Party") shall defend, indemnify, and hold harmless the other party and its officers, directors, employees, and agents (the "Indemnified Parties") from and against any third-party claims, damages, losses, and expenses, including reasonable legal fees, arising out of or relating to the Indemnifying Party''s breach of this Agreement or wilful misconduct.',
    'Mutual indemnity limited to third-party claims arising from breach or wilful misconduct; avoids broad IP indemnity sweeps.'
),
(
    'lib-termination-001',
    'termination',
    'Either party may terminate this Agreement for convenience upon thirty (30) days'' written notice to the other party. Either party may terminate this Agreement immediately upon written notice if the other party materially breaches this Agreement and fails to cure such breach within fifteen (15) days of receiving written notice of the breach.',
    'Standard 30-day convenience termination with 15-day cure period for material breach.'
),
(
    'lib-confidentiality-001',
    'confidentiality',
    '"Confidential Information" means any non-public information disclosed by one party (the "Disclosing Party") to the other (the "Receiving Party") that is designated as confidential or that reasonably should be understood to be confidential given the nature of the information and circumstances of disclosure. The Receiving Party shall use the Confidential Information solely for the purposes of this Agreement and shall not disclose it to any third party without prior written consent, except to its employees or contractors who need to know it for the purposes of this Agreement and who are bound by obligations no less restrictive than those contained herein.',
    'Standard bilateral NDA clause; excludes public domain, independently developed, and legally compelled disclosures.'
),
(
    'lib-ip-ownership-001',
    'ip_ownership',
    'All intellectual property rights in any work product, deliverables, or materials created by the Supplier specifically for the Customer under this Agreement ("Work Product") shall, upon full payment of all applicable fees, vest in and be assigned to the Customer. The Supplier retains ownership of its pre-existing intellectual property and any general tools, methods, or know-how developed independently of this Agreement.',
    'Work-for-hire assignment with carve-out for supplier pre-existing IP; requires full payment as condition.'
),
(
    'lib-governing-law-001',
    'governing_law',
    'This Agreement shall be governed by and construed in accordance with the laws of England and Wales, without regard to its conflict of laws provisions. The parties irrevocably submit to the exclusive jurisdiction of the courts of England and Wales for the resolution of any dispute arising out of or in connection with this Agreement.',
    'English law and exclusive English court jurisdiction; replace with applicable jurisdiction as needed.'
);
